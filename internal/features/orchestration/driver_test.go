package orchestration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// fakeRunner records the subcommands the driver would run, executing none.
type fakeRunner struct{ calls [][]string }

func (r *fakeRunner) run(label string, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	return "", nil
}

// usageRunner behaves like fakeRunner but simulates a real spawn's side
// effect: each "spawn" call creates a fresh RunsDir entry carrying a
// usage.txt with a real token actual, the way execRunner's child processes do
// via writeUsage. This is what lets a test exercise runCycle's real
// RunsTokensSince accounting instead of a hand-fed token count.
type usageRunner struct {
	fakeRunner
	w              *workspace.Workspace
	tokensPerSpawn int
}

func (r *usageRunner) run(label string, args ...string) (string, error) {
	r.fakeRunner.run(label, args...)
	if len(args) > 0 && args[0] == "spawn" {
		runDir := r.w.RunDir(ulid.New())
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return "", err
		}
		body := fmt.Sprintf("output_tokens: %d\ninput_tokens: 0\nnum_turns: 1\ncost_usd: 0\n", r.tokensPerSpawn)
		if err := os.WriteFile(filepath.Join(runDir, "usage.txt"), []byte(body), 0o644); err != nil {
			return "", err
		}
	}
	return "", nil
}

// filingRunner behaves like fakeRunner but simulates the one real side effect
// the review phase's spawned auditor has on the world: filing a fresh task.
// It fires exactly once, on the first spawn carrying reviewRole, so a test can
// drive an empty backlog through idle → review (task filed) → build without a
// real agent ever running.
type filingRunner struct {
	fakeRunner
	w          *workspace.Workspace
	reviewRole string
	filedRef   string
}

func (r *filingRunner) run(label string, args ...string) (string, error) {
	r.fakeRunner.run(label, args...)
	if r.filedRef == "" && len(args) > 0 && args[0] == "spawn" && contains(args, r.reviewRole) {
		t, err := store.CreateTask(r.w, "a-root", "p", "Follow-up filed by review", store.TaskOpts{Accept: []string{"a"}})
		if err != nil {
			return "", err
		}
		r.filedRef = fmt.Sprintf("%03d", t.Seq)
	}
	return "", nil
}

func (r *fakeRunner) firstArgs() []string {
	var out []string
	for _, c := range r.calls {
		if len(c) > 0 {
			out = append(out, c[0])
		}
	}
	return out
}

func loopEnv(t *testing.T) *workspace.Workspace {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("DACLI_AGENT", "")
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "x@x"}, {"config", "user.name", "x"},
		{"checkout", "-q", "-b", "main"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

func newDriver(w *workspace.Workspace, r runner, gov *Governor) *driver {
	return &driver{
		ctx:   &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root},
		w:     w,
		cfg:   loopCfg{project: "p", implRole: "fixer", reviewRole: "go-auditor", width: 2, pr: true},
		gov:   gov,
		run:   r,
		sleep: func(time.Duration) {},
		now:   func() time.Time { return time.Unix(1_000_000, 0) },
	}
}

func TestDriverRunsSprintPhasesInOrder(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	got := strings.Join(fr.firstArgs(), ",")
	// build spawn → wait → ship → review spawn → retro
	for _, want := range []string{"spawn", "wait", "ship", "retro"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected phase %q in sequence, got: %s", want, got)
		}
	}
	// The build spawn must target the ready task with the implementer role + PR.
	var buildSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "fixer") {
			buildSpawn = c
		}
	}
	if buildSpawn == nil {
		t.Fatal("no build spawn with the impl role")
	}
	for _, need := range []string{"--task", "--role", "fixer", "--detach", "--pr"} {
		if !contains(buildSpawn, need) {
			t.Fatalf("build spawn missing %q: %v", need, buildSpawn)
		}
	}
	// The review phase must spawn the reviewer role.
	sawReview := false
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "go-auditor") {
			sawReview = true
		}
	}
	if !sawReview {
		t.Fatal("review phase did not spawn the reviewer role")
	}
}

func TestDriverIdlesWhenBacklogEmpty(t *testing.T) {
	w := loopEnv(t) // no ready tasks
	fr := &fakeRunner{}
	// Idle path with dry-run stops after one pass so the test terminates.
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3, Idle: time.Millisecond})
	d.cfg.dryRun = true
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}
	// No build spawn should have happened; only the review-regeneration spawn.
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "fixer") {
			t.Fatalf("idle cycle must not spawn implementers, got: %v", c)
		}
	}
}

// TestDriverIdleReviewFilesTaskThenBuilds is the 097 regression: an empty
// backlog must go through a real idle→review→build transition, not just idle
// forever. The idle cycle's review phase (simulated here by filingRunner
// standing in for the auditor's `task add`) files the first task; the loop
// must then pick that task up as ready backlog on its very next pass and run
// a build cycle for it — with no real process ever spawned.
func TestDriverIdleReviewFilesTaskThenBuilds(t *testing.T) {
	w := loopEnv(t) // no tasks at all — empty backlog
	fr := &filingRunner{w: w, reviewRole: "go-auditor"}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3, Idle: time.Millisecond})
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	if fr.filedRef == "" {
		t.Fatal("idle cycle's review phase never filed a task")
	}

	// The first spawn overall must be the idle cycle's review spawn — no
	// builder should run before there is anything ready to build.
	var firstSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" {
			firstSpawn = c
			break
		}
	}
	if firstSpawn == nil || !contains(firstSpawn, "go-auditor") {
		t.Fatalf("expected the idle cycle's first spawn to be the review role, got: %v", firstSpawn)
	}

	// Once review filed a task, the loop must build it: a fixer spawn
	// targeting exactly that task's ref.
	var buildSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "fixer") {
			buildSpawn = c
		}
	}
	if buildSpawn == nil {
		t.Fatal("no build spawn followed the filed task — idle never transitioned to build")
	}
	if !contains(buildSpawn, "--task") || !contains(buildSpawn, fr.filedRef) {
		t.Fatalf("build spawn must target the filed task %s, got: %v", fr.filedRef, buildSpawn)
	}
}

func commitTo(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, a := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", name}} {
		c := exec.Command("git", a...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", a, err, out)
		}
	}
}

// TestTrunkMarkerReflectsTrunkAdvance is the regression for the thrash guard's
// progress signal: `landed` must track commits that actually reach trunk, not a
// task-status delta (which counts a proposed-but-unmerged PR as progress and
// lets a --pr --auto loop that never lands anything dodge NoProgressHalt). A
// real commit on trunk moves the marker by exactly one; a cycle that merges
// nothing leaves it flat.
func TestTrunkMarkerReflectsTrunkAdvance(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt") // ensure a born trunk branch
	d := newDriver(w, &fakeRunner{}, &Governor{})
	d.trunkBranch = d.resolveTrunkBranch()

	before := d.trunkMarker()
	if flat := d.trunkMarker(); flat != before {
		t.Fatalf("marker must be stable when trunk does not move: %d vs %d", before, flat)
	}
	commitTo(t, w.Root, "landed.txt")
	if after := d.trunkMarker(); after != before+1 {
		t.Fatalf("marker delta want +1 after a trunk commit, got before=%d after=%d", before, after)
	}
}

// TestRunCycleSumsRealUsageTokensAndGovernorSleeps is the 091 regression: a
// cycle's charge must come from the ACTUAL usage.txt written by the runs it
// spawned (build + review), not a caller-supplied number, and that real charge
// must be able to trip the window governor — otherwise --window-tokens is a
// no-op no matter what the Governor unit tests show in isolation.
func TestRunCycleSumsRealUsageTokensAndGovernorSleeps(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	const tokensPerSpawn = 500 // build spawn + review spawn == 1000, well over the window below
	ur := &usageRunner{w: w, tokensPerSpawn: tokensPerSpawn}
	gov := &Governor{WindowDur: time.Hour, WindowTokens: 100}
	d := newDriver(w, ur, gov)
	d.cfg.width = 1

	tokens := d.runCycle([]*store.Task{task})
	if tokens < 2*tokensPerSpawn {
		t.Fatalf("want runCycle to sum real per-cycle usage.txt actuals (>= %d from 2 spawns), got %d",
			2*tokensPerSpawn, tokens)
	}

	if dec, why := gov.AfterCycle(0, tokens); dec == Halt {
		t.Fatalf("AfterCycle should not halt here, got %s (%s)", dec, why)
	}
	dec, why := gov.Before(1, time.Unix(1_000_000, 0))
	if dec != SleepWindow {
		t.Fatalf("want SleepWindow once the real per-cycle charge (%d) exceeds the window budget (%d), got %s (%s)",
			tokens, gov.WindowTokens, dec, why)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
