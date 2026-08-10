package orchestration

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// nonGitLoopEnv is loopEnv without the `git init`: a workspace whose root is
// not a repository at all, so every git op the driver attempts fails. It
// stands in for the transient faults 212 is about (an index lock, a timed-out
// rev-list, git briefly unavailable) — the loop must be able to tell "could
// not measure" from "measured zero".
func nonGitLoopEnv(t *testing.T) *workspace.Workspace {
	t.Helper()
	unsetAgentEnv(t)
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

// stopFileRunner writes the operator's stop file on its first spawn — a child
// agent (or the operator) touching STOP while a wave is in flight.
type stopFileRunner struct {
	fakeRunner
	stopFile string
	spawns   int
}

func (r *stopFileRunner) run(label string, args ...string) (string, error) {
	r.fakeRunner.run(label, args...)
	if len(args) > 0 && args[0] == "spawn" {
		r.spawns++
		if err := os.WriteFile(r.stopFile, []byte("stop"), 0o644); err != nil {
			return "", err
		}
	}
	return "", nil
}

// ---------------------------------------------------------------- 207 -----

// TestGovernorStopFileIsCheckedMidWave is the 207 regression: the stop file is
// the operator's only kill switch, and Before() consults it once per cycle —
// never while a wave of children is running. A STOP that appears after the
// first spawn must stop the loop from launching any further agent, not be
// noticed a whole cycle later.
func TestGovernorStopFileIsCheckedMidWave(t *testing.T) {
	w := loopEnv(t)
	for _, title := range []string{"A", "B", "C"} {
		if _, err := store.CreateTask(w, "a-root", "p", title, store.TaskOpts{Accept: []string{"a"}}); err != nil {
			t.Fatal(err)
		}
	}
	ready, err := readyTasks(w, "p")
	if err != nil {
		t.Fatal(err)
	}

	stop := filepath.Join(t.TempDir(), "STOP")
	r := &stopFileRunner{stopFile: stop}
	d := newDriver(w, r, &Governor{StopFile: stop})
	d.cfg.width = 3

	d.runCycle(ready)

	if r.spawns != 1 {
		t.Fatalf("want exactly 1 spawn — the stop file appeared during the first one, got %d", r.spawns)
	}
	for _, c := range r.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "go-auditor") {
			t.Fatal("review agent spawned after the stop file appeared — no new agent may start once STOP exists")
		}
	}
}

// TestGovernorStopLatchesAgainstDeletion is the other half of 207: STOP lives
// inside the very repo the child agents edit, so a child can delete it. Once
// the governor has seen it, the halt must stick — resuming is a restart, an
// operator affordance, not something a governed child can grant itself.
func TestGovernorStopLatchesAgainstDeletion(t *testing.T) {
	stop := filepath.Join(t.TempDir(), "STOP")
	g := &Governor{StopFile: stop}
	if err := os.WriteFile(stop, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if d, _ := g.Before(1, time.Unix(0, 0)); d != Halt {
		t.Fatalf("want Halt with the stop file present, got %s", d)
	}
	if err := os.Remove(stop); err != nil {
		t.Fatal(err)
	}
	if d, _ := g.Before(1, time.Unix(0, 0)); d != Halt {
		t.Fatalf("want Halt to LATCH after the stop file was deleted, got %s — a child deleting STOP must not revoke the kill switch", d)
	}
}

// TestGovernorStateWriteIsAtomic is the 207 durability half: the governor
// snapshot is what makes the token ceiling and thrash streak survive a
// restart, and os.WriteFile truncates in place — a crash (or a reader) mid-
// write sees a torn file. A temp+rename write replaces the file wholesale, so
// the path never names a partially written file.
func TestGovernorStateWriteIsAtomic(t *testing.T) {
	w := loopEnv(t)
	writeGovernorState(w, "p", governorState{Cycle: 1, WindowStart: time.Unix(1_000_000, 0).UTC(), WindowSpent: 10})
	first, err := os.Stat(governorStateFile(w, "p"))
	if err != nil {
		t.Fatal(err)
	}
	writeGovernorState(w, "p", governorState{Cycle: 2, WindowStart: time.Unix(1_000_000, 0).UTC(), WindowSpent: 20})
	second, err := os.Stat(governorStateFile(w, "p"))
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(first, second) {
		t.Fatal("governor state was rewritten in place — a truncate-then-write leaves a window where a reader sees a torn file; write temp+rename")
	}
	// No temp files may be orphaned in the state directory.
	ents, err := os.ReadDir(filepath.Dir(governorStateFile(w, "p")))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".dacli-tmp-") {
			t.Fatalf("atomic write orphaned a temp file: %s", e.Name())
		}
	}
}

// TestGovernorStateReadRejectsMalformed is the 207 validation half: the file
// is plain `key: value` text inside a repo the governed children edit, and
// every field was parsed with the error discarded — so a torn read, or a child
// writing `window_spent: 0`, silently restored ZEROES for exactly the counters
// that were persisted to survive a restart. A snapshot that does not parse
// must be an error, not a zero-valued governorState.
func TestGovernorStateReadRejectsMalformed(t *testing.T) {
	cases := map[string]string{
		"torn write":            "cycle: 12\nwindow_st",
		"missing window_spent":  "cycle: 12\nwindow_start: 2026-01-01T00:00:00Z\nzero_streak: 1\n",
		"unparseable cycle":     "cycle: twelve\nwindow_start: 2026-01-01T00:00:00Z\nwindow_spent: 5\nzero_streak: 1\n",
		"negative window_spent": "cycle: 12\nwindow_start: 2026-01-01T00:00:00Z\nwindow_spent: -5\nzero_streak: 1\n",
		"negative zero_streak":  "cycle: 12\nwindow_start: 2026-01-01T00:00:00Z\nwindow_spent: 5\nzero_streak: -1\n",
		"unparseable time":      "cycle: 12\nwindow_start: yesterday\nwindow_spent: 5\nzero_streak: 1\n",
		"future window_start":   "cycle: 12\nwindow_start: 2999-01-01T00:00:00Z\nwindow_spent: 5\nzero_streak: 1\n",
		"empty file":            "",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			w := loopEnv(t)
			path := governorStateFile(w, "p")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			st, err := readGovernorState(w, "p")
			if err == nil {
				t.Fatalf("want an error for a malformed snapshot, got %+v — silently zeroing resets the very guards this file exists to preserve", st)
			}
			if !errors.Is(err, errCorruptState) {
				t.Fatalf("want errCorruptState so the caller can tell corruption from a missing file, got %v", err)
			}
		})
	}
}

// TestLoopRefusesCorruptGovernorState closes the 207 loop at the command
// surface: `dacli loop` reloading a corrupt snapshot must REFUSE, not start a
// run with a zeroed budget window and thrash streak — that is precisely the
// state a child would forge to defeat the token ceiling.
func TestLoopRefusesCorruptGovernorState(t *testing.T) {
	w := loopEnv(t)
	path := governorStateFile(w, "p")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("cycle: 4\nwindow_spent: not-a-number\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	err := cmdLoop(ctx, []string{"--project", "p", "--max-cycles", "1", "--dry-run"})
	if err == nil {
		t.Fatal("want a refusal when the persisted governor state is corrupt, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("refusal must name the file the operator has to inspect, got: %v", err)
	}
}

// ---------------------------------------------------------------- 218 -----

// TestGovernorZeroWindowDoesNotDisableBudget is the 218 regression: with
// WindowDur == 0, `now.Sub(windowStart) >= WindowDur` is true on EVERY call,
// so the window rolled and windowSpent reset before it was ever compared to
// WindowTokens — a token budget that silently disabled itself.
func TestGovernorZeroWindowDoesNotDisableBudget(t *testing.T) {
	base := time.Unix(1_000_000, 0)
	g := &Governor{WindowTokens: 100} // WindowDur deliberately 0
	if d, _ := g.Before(1, base); d != Proceed {
		t.Fatalf("first cycle should proceed, got %s", d)
	}
	g.AfterCycle(1, 150) // overspend the budget
	if d, why := g.Before(1, base.Add(time.Minute)); d != SleepWindow {
		t.Fatalf("want SleepWindow — a zero --budget-window must not disable --window-tokens, got %s (%s)", d, why)
	}
	if g.WindowSpent() != 150 {
		t.Fatalf("want the spend preserved across the check, got %d", g.WindowSpent())
	}
	if rem := g.WindowRemaining(base.Add(time.Minute)); rem <= 0 {
		t.Fatalf("want a real reset horizon for the defaulted window, got %s", rem)
	}
}

// TestLoopRejectsZeroBudgetWindowWithTokenBudget is 218 at the flag surface:
// `--budget-window 0` parses cleanly, so an operator who asked for a token
// ceiling would silently get none. Refuse the combination rather than run
// unbudgeted.
func TestLoopRejectsZeroBudgetWindowWithTokenBudget(t *testing.T) {
	w := loopEnv(t)
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	err := cmdLoop(ctx, []string{"--project", "p", "--max-cycles", "1", "--dry-run",
		"--window-tokens", "1000", "--budget-window", "0"})
	if err == nil {
		t.Fatal("want a refusal for --window-tokens with a zero --budget-window, got nil")
	}
	if !strings.Contains(err.Error(), "budget-window") {
		t.Fatalf("refusal must name the offending flag, got: %v", err)
	}
}

// TestLoopAcceptsZeroBudgetWindowWithoutTokenBudget guards the other
// direction: with no token ceiling the window is meaningless, so a zero window
// must not turn an ordinary run into a refusal.
func TestLoopAcceptsZeroBudgetWindowWithoutTokenBudget(t *testing.T) {
	w := loopEnv(t)
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdLoop(ctx, []string{"--project", "p", "--max-cycles", "1", "--dry-run", "--budget-window", "0"}); err != nil {
		t.Fatalf("a zero budget window with no token budget must be fine, got: %v", err)
	}
}

// ---------------------------------------------------------------- 211 -----

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v (in %s): %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestResolveTrunkBranchNeverReturnsHEAD is the 211 regression: on a detached
// HEAD `rev-parse --abbrev-ref HEAD` returns the literal string "HEAD", which
// the resolver handed on as a branch name — trunkMarker then counted
// `origin HEAD` and syncTrunk merged an arbitrary ref. Whatever it returns
// must be a real branch, or nothing.
func TestResolveTrunkBranchNeverReturnsHEAD(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt")
	gitIn(t, w.Root, "checkout", "-q", "--detach", "HEAD")

	d := newDriver(w, &fakeRunner{}, &Governor{})

	// Detached, but a real trunk still exists: it must be found.
	if got := d.resolveTrunkBranch(); got != "main" {
		t.Fatalf("detached HEAD over an existing main: want main, got %q", got)
	}

	// Detached with no trunk branch at all: the old fallthrough produced "HEAD".
	gitIn(t, w.Root, "branch", "-D", "main")
	got := d.resolveTrunkBranch()
	if got == "HEAD" {
		t.Fatal(`resolveTrunkBranch returned the literal "HEAD" — that is not a branch, and trunk measurement/sync against it is meaningless`)
	}
	if got != "" {
		t.Fatalf("with no resolvable trunk the honest answer is empty, got %q", got)
	}
}

// TestResolveTrunkBranchPrefersOriginOverStaleLocalBranch is the other 211
// half: with origin/HEAD unset (the CI and shallow-clone default) the resolver
// fell straight through to a local `main` that nothing lands on, so progress
// was measured against a branch the team abandoned. Origin's branch is the
// trunk; the local leftover is not.
func TestResolveTrunkBranchPrefersOriginOverStaleLocalBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	origin := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "master", origin).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}

	w := loopEnv(t) // local checkout is on `main`
	commitTo(t, w.Root, "seed.txt")
	gitIn(t, w.Root, "remote", "add", "origin", origin)
	// Real trunk lives on origin/master; `main` is the stale local leftover.
	gitIn(t, w.Root, "push", "-q", "origin", "main:master")
	gitIn(t, w.Root, "fetch", "-q", "origin")
	if gitIn(t, w.Root, "rev-parse", "--verify", "refs/remotes/origin/master") == "" {
		t.Fatal("test setup: origin/master remote-tracking ref missing")
	}
	// Unset origin/HEAD — the CI and shallow-clone default, and the exact
	// condition under which the resolver fell through to the local leftover.
	unset := exec.Command("git", "symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
	unset.Dir = w.Root
	unset.CombinedOutput() //nolint:errcheck // absent already is the state we want
	if out, err := exec.Command("git", "-C", w.Root, "rev-parse", "--abbrev-ref", "origin/HEAD").CombinedOutput(); err == nil {
		t.Fatalf("test setup: origin/HEAD must be unset for this case, still resolves to %s", strings.TrimSpace(string(out)))
	}

	d := newDriver(w, &fakeRunner{}, &Governor{})
	if got := d.resolveTrunkBranch(); got != "master" {
		t.Fatalf("want the trunk origin actually carries (master), got %q — a stale local branch is not the trunk", got)
	}
}

// ---------------------------------------------------------------- 212 -----

// TestTrunkMarkerReportsMeasurementFailure is the 212 regression at the
// measurement itself: when every rev-list variant fails the marker returned 0,
// which is indistinguishable from a genuinely empty trunk.
func TestTrunkMarkerReportsMeasurementFailure(t *testing.T) {
	w := nonGitLoopEnv(t)
	d := newDriver(w, &fakeRunner{}, &Governor{})
	d.cfg.dryRun = true // skip the network fetch; the local rev-lists still fail
	if _, ok := d.trunkMarker(); ok {
		t.Fatal("want ok=false when trunk cannot be measured — a failed measurement must not read as zero progress")
	}

	g := loopEnv(t)
	commitTo(t, g.Root, "seed.txt")
	dg := newDriver(g, &fakeRunner{}, &Governor{})
	dg.cfg.dryRun = true
	dg.trunkBranch = dg.resolveTrunkBranch()
	if n, ok := dg.trunkMarker(); !ok || n <= 0 {
		t.Fatalf("want a real measurement on a real repo, got n=%d ok=%v", n, ok)
	}
}

// TestGovernorUnmeasuredCycleLeavesThrashStreakAlone is 212 in the governor: a
// cycle whose trunk advancement could not be measured must not feed the thrash
// guard a fabricated zero — nor reset it.
func TestGovernorUnmeasuredCycleLeavesThrashStreakAlone(t *testing.T) {
	g := &Governor{NoProgressHalt: 2}
	g.AfterCycle(0, 10) // one real zero-progress cycle: streak 1
	// The guard is a single condition on purpose. It was `d != Halt &&
	// g.ZeroStreak() != 1`, and a conjunction can only make a Fatalf WEAKER:
	// the regression this line names — AfterCycleUnmeasured treating the cycle
	// as a real zero-progress one — increments the streak to 2, which equals
	// NoProgressHalt and therefore returns Halt. `d != Halt` would then be
	// false, the conjunction false, and the line silent on exactly the bug it
	// is named for.
	if d, why := g.AfterCycleUnmeasured(10); g.ZeroStreak() != 1 {
		t.Fatalf("unmeasured cycle must leave the streak at 1, got %d (%s %s)", g.ZeroStreak(), d, why)
	}
	if d, _ := g.AfterCycleUnmeasured(10); d == Halt {
		t.Fatal("an unmeasured cycle must never trip the thrash guard on its own")
	}
	if g.ZeroStreak() != 1 {
		t.Fatalf("want the streak untouched by unmeasured cycles, got %d", g.ZeroStreak())
	}
	if g.Cycle() != 3 {
		t.Fatalf("unmeasured cycles still happened and must still count: want cycle 3, got %d", g.Cycle())
	}
	if g.WindowSpent() != 30 {
		t.Fatalf("unmeasured cycles still burn tokens: want 30, got %d", g.WindowSpent())
	}
}

// TestLoopDoesNotCountUnmeasurableTrunkAsZeroProgress is the 212 end-to-end
// regression: a cycle that could not measure trunk used to compute
// `landed = 0 - prevTrunk` → clamped to 0 → zeroStreak++, and then set
// prevTrunk = 0 so the NEXT cycle read the whole repo history as progress.
// A loop whose git is unavailable must halt on nothing at all here.
func TestLoopDoesNotCountUnmeasurableTrunkAsZeroProgress(t *testing.T) {
	w := nonGitLoopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	gov := &Governor{MaxCycles: 1, NoProgressHalt: 1}
	d := newDriver(w, &fakeRunner{}, gov)
	d.cfg.dryRun = true
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}
	if gov.ZeroStreak() != 0 {
		t.Fatalf("an unmeasurable trunk must not increment the thrash streak, got %d", gov.ZeroStreak())
	}
	if gov.Cycle() != 1 {
		t.Fatalf("the cycle still ran and must still be counted, got %d", gov.Cycle())
	}
}
