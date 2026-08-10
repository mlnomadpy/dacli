package orchestration

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func noRemoteRepo(t *testing.T) *workspace.Workspace {
	t.Helper()
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "-Af", "--", ".dacli")
	gitAt(t, dir, "commit", "-q", "-m", "base")
	return w
}

// With no `origin`, the PR path cannot land anything: gh fails on every call,
// every branch reads "unknown", no accept resolves, and the loop re-picks the
// same tasks forever. That is issue #382's first symptom, and the prevention
// is to not choose the PR path at all when there is no remote.
func TestLoopLandsLocallyWhenThereIsNoRemote(t *testing.T) {
	w := noRemoteRepo(t)
	f, err := clikit.ParseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	if prMode(w, f) {
		t.Error("the loop chose the PR path with no origin — every landing check would dead-end and no task would ever close")
	}

	// An explicit --no-pr is unchanged.
	off, _ := clikit.ParseFlags([]string{"--no-pr"})
	if prMode(w, off) {
		t.Error("--no-pr must always disable the PR path")
	}
}

// A merge into the LOCAL trunk is a real landing. Reporting it as "unknown"
// is what left tasks open forever in a no-remote workspace.
func TestLandStatusSeesALocalMerge(t *testing.T) {
	w := noRemoteRepo(t)
	d := &driver{w: w, trunkBranch: "main", cfg: loopCfg{project: "core"}}

	const branch = "dacli/001-a-task"
	gitAt(t, w.Root, "checkout", "-q", "-b", branch)
	gitAt(t, w.Root, "commit", "-q", "--allow-empty", "-m", "the work")
	gitAt(t, w.Root, "checkout", "-q", "main")

	if got := d.prLandStatus(branch); got != "orphaned" {
		t.Errorf("before the merge: got %q, want orphaned (work exists, not in trunk)", got)
	}

	gitAt(t, w.Root, "merge", "-q", "--no-ff", "-m", "merge", branch)
	if got := d.prLandStatus(branch); got != "merged" {
		t.Errorf("after a LOCAL merge: got %q, want merged — a local landing is a landing, and calling it unknown is what made the loop thrash", got)
	}
}

// A branch that never committed must not read as merged just because it is
// trivially an ancestor of trunk — that is the path that force-accepts empty
// work as done (dacli 168), and it must keep holding with no remote.
func TestLandStatusStillRefusesAnEmptyBranchLocally(t *testing.T) {
	w := noRemoteRepo(t)
	d := &driver{w: w, trunkBranch: "main", cfg: loopCfg{project: "core"}}

	const branch = "dacli/002-empty"
	gitAt(t, w.Root, "branch", branch)
	if got := d.prLandStatus(branch); got != "orphaned" {
		t.Errorf("an empty branch got %q, want orphaned — a spawn that never committed is not a landing", got)
	}
}

// hasOrigin is the whole distinction between "the PR has not merged yet" and
// "there is no GitHub in this picture" — two states that used to collapse into
// the one value that never resolves.
func TestHasOriginDistinguishesNoRemote(t *testing.T) {
	w := noRemoteRepo(t)
	d := &driver{w: w}
	if d.hasOrigin() {
		t.Error("a repo with no remotes reported an origin")
	}
	gitAt(t, w.Root, "remote", "add", "origin", "https://example.invalid/x.git")
	if !d.hasOrigin() {
		t.Error("an origin was added but not detected")
	}
}

// Both spellings of the idle-halt flag reach the same setting. The original
// name asks the reader to hold a double negative — "--no-progress-halt 2"
// reads as "do not halt", so passing a number feels wrong even though it is
// required — and a script that guessed the boolean form died instantly on a
// redirected log, looking exactly like a loop with nothing to do (issue #421).
func TestIdleHaltAcceptsBothSpellings(t *testing.T) {
	for _, name := range []string{"halt-after-idle", "no-progress-halt"} {
		f, err := clikit.ParseFlags([]string{"--" + name, "7"})
		if err != nil {
			t.Fatalf("--%s: %v", name, err)
		}
		got, err := f.IntAliased(3, "halt-after-idle", "no-progress-halt")
		if err != nil || got != 7 {
			t.Errorf("--%s 7 => %d, %v; want 7", name, got, err)
		}
	}

	// Absent, the default still applies — the loop must keep its stop
	// condition when nobody names one.
	f, _ := clikit.ParseFlags(nil)
	if got, err := f.IntAliased(3, "halt-after-idle", "no-progress-halt"); err != nil || got != 3 {
		t.Errorf("default => %d, %v; want 3", got, err)
	}

	// The bare form is refused rather than silently read as a boolean, and the
	// refusal names what it needs.
	bare, _ := clikit.ParseFlags([]string{"--halt-after-idle"})
	if _, err := bare.IntAliased(3, "halt-after-idle", "no-progress-halt"); err == nil {
		t.Error("the bare flag must not be accepted as a boolean")
	} else if !strings.Contains(err.Error(), "integer") {
		t.Errorf("the refusal must name the value it needs, got %v", err)
	}
}

// A sprint integrates a batch of related work onto its own branch and takes
// ONE pull request to main at the end, instead of one PR per fix. Without
// --into the loop always resolved main, so the moment the checkout was on the
// sprint branch it refused every step — "refusing to operate on the wrong
// branch" — and the workflow was unusable (dacli 332).
func TestIntoOverridesTheResolvedTrunk(t *testing.T) {
	w := noRemoteRepo(t) // has a real `main`, which resolution would otherwise pick
	gitAt(t, w.Root, "branch", "sprint/1")

	d := &driver{w: w, cfg: loopCfg{project: "core", into: "sprint/1"}}
	if got := d.resolveTrunkBranch(); got != "sprint/1" {
		t.Errorf("resolveTrunkBranch = %q; --into must win over the resolved trunk", got)
	}

	// And the resolved branch is what ship is actually told to land onto —
	// otherwise the override would be cosmetic.
	d.trunkBranch = d.resolveTrunkBranch()
	args := strings.Join(d.shipArgs("--project", "core"), " ")
	if !strings.Contains(args, "--into sprint/1") {
		t.Errorf("ship args %q must carry --into sprint/1", args)
	}

	// Absent --into, resolution is unchanged: this must not make every repo
	// depend on a flag nobody passes.
	plain := &driver{w: w, cfg: loopCfg{project: "core"}}
	if got := plain.resolveTrunkBranch(); got != "main" {
		t.Errorf("without --into the trunk must still resolve normally, got %q", got)
	}
}

// A typo in --into is a usage error, caught before any agent is spawned. It is
// threaded into every ship and integrate call, so an unvalidated one surfaces
// deep inside a cycle that has already spent tokens, as what looks like a git
// problem rather than a flag problem.
func TestIntoRefusesAnUnknownBranchUpFront(t *testing.T) {
	w := noRemoteRepo(t)
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}

	err := cmdLoop(ctx, []string{"--project", "core", "--into", "no-such-branch", "--max-cycles", "1", "--dry-run"})
	if err == nil {
		t.Fatal("an unknown --into branch must be refused")
	}
	if got := clikit.ExitCode(err); got != 2 {
		t.Errorf("exit %d, want 2 (usage): %v", got, err)
	}
	if !strings.Contains(err.Error(), "no-such-branch") {
		t.Errorf("the refusal must name the branch, got %v", err)
	}
}
