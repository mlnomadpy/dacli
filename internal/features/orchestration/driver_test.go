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
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
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

// unsetAgentEnv clears DACLI_AGENT for the test, restoring whatever the
// process started with. t.Setenv cannot unset a variable, and since dacli 288
// a present-but-empty DACLI_AGENT is a lost token that fails closed rather
// than resolving to root — so a test wanting the root identity must remove
// the variable entirely, not blank it.
func unsetAgentEnv(t *testing.T) {
	t.Helper()
	if v, ok := os.LookupEnv("DACLI_AGENT"); ok {
		t.Setenv("DACLI_AGENT", v)
		_ = os.Unsetenv("DACLI_AGENT")
	}
}

func loopEnv(t *testing.T) *workspace.Workspace {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	unsetAgentEnv(t)
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

// TestLoopBuildsHighestPriorityReadyTaskNotLowestSeq is the 103 regression:
// the BUILD phase must pick the ready frontier's highest MoSCoW-priority task,
// not simply the lowest Seq. A low-seq could filed before a high-seq must
// must NOT be built first — the must (however late it was filed) must win at
// width=1.
func TestLoopBuildsHighestPriorityReadyTaskNotLowestSeq(t *testing.T) {
	w := loopEnv(t)
	could, err := store.CreateTask(w, "a-root", "p", "Low priority, filed first", store.TaskOpts{Priority: "could", Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	must, err := store.CreateTask(w, "a-root", "p", "Critical, filed second", store.TaskOpts{Priority: "must", Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if could.Seq >= must.Seq {
		t.Fatalf("test setup: expected the could task to have the lower seq, got could=%d must=%d", could.Seq, must.Seq)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	d.cfg.width = 1
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	var buildSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "fixer") {
			buildSpawn = c
		}
	}
	if buildSpawn == nil {
		t.Fatal("no build spawn with the impl role")
	}
	mustRef := fmt.Sprintf("%03d", must.Seq)
	couldRef := fmt.Sprintf("%03d", could.Seq)
	if !contains(buildSpawn, mustRef) {
		t.Fatalf("width=1 build must target the higher-priority must task %s, got: %v", mustRef, buildSpawn)
	}
	if contains(buildSpawn, couldRef) {
		t.Fatalf("width=1 build must not target the lower-priority, lower-seq could task %s, got: %v", couldRef, buildSpawn)
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

	before, ok := d.trunkMarker()
	if !ok {
		t.Fatal("trunk must be measurable in a real repo")
	}
	if flat, _ := d.trunkMarker(); flat != before {
		t.Fatalf("marker must be stable when trunk does not move: %d vs %d", before, flat)
	}
	commitTo(t, w.Root, "landed.txt")
	if after, _ := d.trunkMarker(); after != before+1 {
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

	tokens, _ := d.runCycle([]*store.Task{task})
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

// spawnOutcomeRunner simulates per-task spawn outcomes so a test can drive
// runCycle through a mixed batch without any real agent process. The task
// whose ref is refusedRef gets a synchronous spawn error and never gets a
// branch, mirroring a real refusal (taint block, budget, malformed flags);
// every other spawned task gets a real dacli/<seq>-slug branch created, the
// way an implementer's worktree commit would, so runCycle's post-wait branch
// check sees it. "accept" calls really move the named task to done — the same
// effect the real `accept --force` command has — so the test can assert the
// refused task truly stays open, not merely that accept was never invoked
// with its ref.
type spawnOutcomeRunner struct {
	fakeRunner
	w          *workspace.Workspace
	refusedRef string
	refusalMsg string // overrides the default refusal text; "" keeps it generic
}

func (r *spawnOutcomeRunner) run(label string, args ...string) (string, error) {
	r.fakeRunner.run(label, args...)
	switch {
	case len(args) > 0 && args[0] == "spawn":
		ref := argAfter(args, "--task")
		if ref == r.refusedRef {
			msg := r.refusalMsg
			if msg == "" {
				msg = "spawn refused: policy"
			}
			return "", fmt.Errorf("%s", msg)
		}
		t, err := store.FindTask(r.w, ref)
		if err != nil {
			return "", err
		}
		// A real implementer commits work to its branch; the branch is created
		// at spawn (empty) and only carries work once the child commits. Create
		// the branch ONE COMMIT past HEAD so runCycle's branchHasWork check
		// (dacli 168) sees actual work rather than an empty branch — an empty
		// branch is exactly what a failed/killed spawn leaves, and must NOT be
		// treated as built.
		if err := branchWithCommit(r.w.Root, taskBranch(t)); err != nil {
			return "", err
		}
		return "", nil
	case len(args) > 1 && args[0] == "accept":
		t, err := store.FindTask(r.w, args[1])
		if err != nil {
			return "", err
		}
		return "", store.MoveTask(r.w, t, model.StatusDone)
	}
	return "", nil
}

// branchWithCommit creates branch pointing one commit past HEAD without
// touching the working tree — the harness stand-in for an implementer's
// worktree commit, so branchHasWork (dacli 168) sees real work. Uses
// commit-tree plumbing so no checkout is needed.
func branchWithCommit(dir, branch string) error {
	run := func(args ...string) (string, error) {
		c := exec.Command("git", args...)
		c.Dir = dir
		out, err := c.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	tree, err := run("rev-parse", "HEAD^{tree}")
	if err != nil {
		return fmt.Errorf("rev-parse tree: %v (%s)", err, tree)
	}
	parent, err := run("rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("rev-parse HEAD: %v (%s)", err, parent)
	}
	commit, err := run("commit-tree", tree, "-p", parent, "-m", "fixer work")
	if err != nil {
		return fmt.Errorf("commit-tree: %v (%s)", err, commit)
	}
	if out, err := run("branch", branch, commit); err != nil {
		return fmt.Errorf("git branch: %v (%s)", err, out)
	}
	return nil
}

func argAfter(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// TestRunCycleLeavesRefusedSpawnTaskOpenButParksBuiltTaskPending is the 102
// regression, updated for 115: a batch where one task's implementer spawn is
// refused must not be treated as "the whole batch got built" — but the --pr
// LAND step must no longer close the successfully-built sibling's record
// synchronously either. It is parked in pendingAccept (open, not box-checked)
// until reconcilePendingAccepts confirms its PR merged — see
// TestReconcilePendingAccepts*. The refused task gets no such tracking at all.
func TestRunCycleLeavesRefusedSpawnTaskOpenButParksBuiltTaskPending(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt") // a born trunk so `git branch` has a HEAD to point at
	refused, err := store.CreateTask(w, "a-root", "p", "Task whose spawn is refused", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := store.CreateTask(w, "a-root", "p", "Task that builds fine", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	refusedRef := fmt.Sprintf("%03d", refused.Seq)

	r := &spawnOutcomeRunner{w: w, refusedRef: refusedRef}
	d := newDriver(w, r, &Governor{})
	d.cfg.width = 2

	d.runCycle([]*store.Task{refused, ok})

	for _, c := range r.calls {
		if len(c) > 1 && c[0] == "accept" {
			t.Fatalf("accept --force must never be called synchronously in the LAND step — it awaits merge confirmation: %v", c)
		}
	}

	stillOpen, err := store.ListTasks(w, "p", model.StatusOpen)
	if err != nil {
		t.Fatal(err)
	}
	foundOpen := map[int]bool{}
	for _, tk := range stillOpen {
		foundOpen[tk.Seq] = true
	}
	if !foundOpen[refused.Seq] {
		t.Fatalf("task %s whose spawn was refused must remain open for the next cycle to re-pick", refusedRef)
	}
	if !foundOpen[ok.Seq] {
		t.Fatalf("successfully built task %03d must remain open until its PR's merge is confirmed", ok.Seq)
	}

	pending := map[int]bool{}
	for _, p := range d.pendingAccept {
		pending[p.Seq] = true
	}
	if pending[refused.Seq] {
		t.Fatalf("refused task %s must never be tracked as pending accept: %v", refusedRef, d.pendingAccept)
	}
	if !pending[ok.Seq] {
		t.Fatalf("successfully built task %03d must be tracked as pending accept, got: %v", ok.Seq, d.pendingAccept)
	}
}

// stubOrchestrationGH swaps the package's runGH var for the duration of the
// test and restores it afterward.
func stubOrchestrationGH(t *testing.T, fn func(dir string, args ...string) (string, error)) {
	t.Helper()
	orig := runGH
	runGH = fn
	t.Cleanup(func() { runGH = orig })
}

// TestReconcilePendingAcceptsClosesOnConfirmedMerge is the 115 acceptance
// case: once gh reports the pending task's PR MERGED, reconcilePendingAccepts
// must close the task record now (accept --force) and drop it from tracking —
// this is the only point the backlog is now allowed to claim the task done.
func TestReconcilePendingAcceptsClosesOnConfirmedMerge(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	stubOrchestrationGH(t, func(dir string, args ...string) (string, error) {
		return `[{"state":"MERGED"}]`, nil
	})

	r := &spawnOutcomeRunner{w: w}
	d := newDriver(w, r, &Governor{})
	branch := taskBranch(task)
	d.pendingAccept = []pendingAccept{{Seq: task.Seq, Branch: branch}}

	d.reconcilePendingAccepts()

	if len(d.pendingAccept) != 0 {
		t.Fatalf("pendingAccept should be cleared once merged, got: %v", d.pendingAccept)
	}
	sawAccept := false
	for _, c := range r.calls {
		if len(c) > 1 && c[0] == "accept" && c[1] == fmt.Sprintf("%03d", task.Seq) {
			sawAccept = true
		}
	}
	if !sawAccept {
		t.Fatalf("expected accept --force once the PR is confirmed merged, calls: %v", r.calls)
	}
}

// TestReconcilePendingAcceptsReopensOnClosedUnmergedPR is the falsely-done
// regression this task exists to fix: a PR that CI eventually rejects (closed
// without merging) must NOT leave the task silently done, and must not stay
// stuck pending forever either — it drops from tracking so the still-open
// task re-enters the ready pool for a fresh attempt.
func TestReconcilePendingAcceptsReopensOnClosedUnmergedPR(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	stubOrchestrationGH(t, func(dir string, args ...string) (string, error) {
		return `[{"state":"CLOSED"}]`, nil
	})

	r := &spawnOutcomeRunner{w: w}
	d := newDriver(w, r, &Governor{})
	d.pendingAccept = []pendingAccept{{Seq: task.Seq, Branch: taskBranch(task)}}

	d.reconcilePendingAccepts()

	if len(d.pendingAccept) != 0 {
		t.Fatalf("pendingAccept should drop a closed-unmerged PR, got: %v", d.pendingAccept)
	}
	for _, c := range r.calls {
		if len(c) > 1 && c[0] == "accept" {
			t.Fatalf("accept --force must never be called for a PR that closed unmerged: %v", c)
		}
	}
	open, err := store.ListTasks(w, "p", model.StatusOpen)
	if err != nil {
		t.Fatal(err)
	}
	foundOpen := false
	for _, tk := range open {
		if tk.Seq == task.Seq {
			foundOpen = true
		}
	}
	if !foundOpen {
		t.Fatalf("task %03d must remain open (never closed) after its PR closed unmerged", task.Seq)
	}
}

// TestReconcilePendingAcceptsKeepsWaitingWhilePROpen proves the steady state:
// while gh still reports the PR OPEN with auto-merge queued (no CI verdict yet),
// the task stays parked pending — not closed, not dropped — so a slow-to-land
// CI run is never misread as either success or abandonment.
func TestReconcilePendingAcceptsKeepsWaitingWhilePROpen(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	// OPEN with auto-merge queued: the healthy "landing" path.
	stubOrchestrationGH(t, func(dir string, args ...string) (string, error) {
		return `[{"state":"OPEN","autoMergeRequest":{"enabledAt":"2026-08-04T00:00:00Z"}}]`, nil
	})

	r := &fakeRunner{}
	d := newDriver(w, r, &Governor{})
	d.pendingAccept = []pendingAccept{{Seq: task.Seq, Branch: taskBranch(task)}}

	d.reconcilePendingAccepts()

	if len(d.pendingAccept) != 1 {
		t.Fatalf("a still-open PR must stay pending, got: %v", d.pendingAccept)
	}
	for _, c := range r.calls {
		if len(c) > 0 && c[0] == "accept" {
			t.Fatalf("accept --force must never be called while the PR is still open: %v", c)
		}
	}
}

// TestReconcilePendingAcceptsFlagsStrandedPR is the task-290 loop guard: a PR
// that is OPEN but has NO auto-merge queued (the fixer's `dacli pr --auto`
// failed to queue it) will never self-land. The loop must not silently treat it
// like a queued "landing" PR — it must surface that the PR is stranded so it is
// not counted as still-landing forever. The task stays parked (still open, work
// not on trunk) but the reconcile logs it as needing attention.
func TestReconcilePendingAcceptsFlagsStrandedPR(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	// OPEN but NO autoMergeRequest: auto-merge was never queued.
	stubOrchestrationGH(t, func(dir string, args ...string) (string, error) {
		return `[{"state":"OPEN"}]`, nil
	})

	branch := taskBranch(task)
	r := &fakeRunner{}
	d := newDriver(w, r, &Governor{})
	if got := d.prLandStatus(branch); got != "stranded" {
		t.Fatalf("an OPEN PR with no auto-merge queued must classify as stranded, got %q", got)
	}
	d.pendingAccept = []pendingAccept{{Seq: task.Seq, Branch: branch}}

	d.reconcilePendingAccepts()

	// Kept pending (work is not on trunk) — never accepted as done.
	if len(d.pendingAccept) != 1 {
		t.Fatalf("a stranded PR must stay pending, got: %v", d.pendingAccept)
	}
	for _, c := range r.calls {
		if len(c) > 0 && c[0] == "accept" {
			t.Fatalf("accept --force must never be called for a stranded PR: %v", c)
		}
	}
	// The distinguishing signal: the loop said the PR will NOT self-land.
	if logs := d.ctx.Stdout.(*bytes.Buffer).String(); !strings.Contains(logs, "NOT queued for auto-merge") {
		t.Fatalf("reconcile must flag a stranded PR as not queued for auto-merge, log:\n%s", logs)
	}
}

// TestExcludePendingKeepsInFlightTaskOutOfReadyFrontier is the other half of
// the 115 fix: a task parked in pendingAccept must be excluded from the ready
// frontier, or the very next cycle would rebuild it while its first PR is
// still in flight.
func TestExcludePendingKeepsInFlightTaskOutOfReadyFrontier(t *testing.T) {
	w := loopEnv(t)
	pending, err := store.CreateTask(w, "a-root", "p", "PR in flight", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateTask(w, "a-root", "p", "Untouched task", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	ready := []*store.Task{pending, other}
	filtered := excludePending(ready, []pendingAccept{{Seq: pending.Seq, Branch: taskBranch(pending)}})

	if len(filtered) != 1 || filtered[0].Seq != other.Seq {
		t.Fatalf("expected only the untouched task to remain ready, got: %v", filtered)
	}
}

// TestDriverGitAbortsOnHungSubprocess is the 105 regression: driver.git must
// route through gitx's deadline-bounded runner, not a bare exec.Command, so a
// wedged git child (a credential-helper prompt, a hung index lock) can never
// freeze the perpetual loop. A fake `git` on PATH that just sleeps stands in
// for the hang; gitx.LocalTimeout is shrunk for the duration of the test so
// the assertion does not have to wait out the real 30s deadline.
func TestDriverGitAbortsOnHungSubprocess(t *testing.T) {
	w := loopEnv(t)
	d := newDriver(w, &fakeRunner{}, &Governor{})

	fakeDir := t.TempDir()
	scriptPath := filepath.Join(fakeDir, "git")
	// `exec sleep` (not a plain `sleep` line) replaces the shell's own process
	// image instead of forking a child — so killing this one PID on timeout
	// actually kills the sleeper too, instead of leaving it holding the output
	// pipe open and stalling CombinedOutput() for the full sleep duration.
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexec sleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	orig := gitx.LocalTimeout
	gitx.LocalTimeout = 200 * time.Millisecond
	defer func() { gitx.LocalTimeout = orig }()

	start := time.Now()
	_, err := d.git("status")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the hung git subprocess to return a timeout error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("driver.git did not abort within the deadline; took %s", elapsed)
	}
}

// TestDriverIdleBranchChargesReviewTokensToWindow is the 106 regression: the
// Idle branch's review spawn must charge its real usage.txt tokens to the
// SAME governor window AfterCycle charges — before this fix, Idle never
// reached AfterCycle (the only writer of windowSpent), so an idling loop's
// dominant steady-state cost never counted against --window-tokens.
func TestDriverIdleBranchChargesReviewTokensToWindow(t *testing.T) {
	w := loopEnv(t) // no ready tasks: every Before() call returns Idle
	ur := &usageRunner{w: w, tokensPerSpawn: 42}
	gov := &Governor{MaxCycles: 1, NoProgressHalt: 3, Idle: time.Millisecond, WindowDur: time.Hour, WindowTokens: 1_000_000}
	d := newDriver(w, ur, gov)
	d.cfg.dryRun = true // a single Idle pass, then stop
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}
	if gov.WindowSpent() <= 0 {
		t.Fatalf("want the Idle branch's review spawn to charge the governor window, got WindowSpent=%d", gov.WindowSpent())
	}
}

// TestIdleTicksAccumulateWindowTokensAndTripGuard is the 106 acceptance test:
// repeated Idle ticks — each running a real review spawn and charging its
// actual usage.txt tokens via Governor.ChargeIdleTokens — must accumulate in
// windowSpent, and once that accumulation alone exceeds --window-tokens, the
// window guard (Before → SleepWindow) must trip purely from idle-path spend,
// with no build cycle ever having run.
func TestIdleTicksAccumulateWindowTokensAndTripGuard(t *testing.T) {
	w := loopEnv(t) // no ready tasks
	const tokensPerSpawn = 30
	ur := &usageRunner{w: w, tokensPerSpawn: tokensPerSpawn}
	gov := &Governor{WindowDur: time.Hour, WindowTokens: 100}
	d := newDriver(w, ur, gov)
	now := d.now()

	ticks := 0
	for {
		dec, why := gov.Before(0, now)
		if dec == SleepWindow {
			break
		}
		if dec != Idle {
			t.Fatalf("tick %d: want Idle with an empty backlog, got %s (%s)", ticks, dec, why)
		}
		// ULID run-dir names carry only millisecond resolution plus random
		// bits, so two runs created within the same millisecond are not
		// guaranteed to sort in creation order; a real idle loop spaces
		// ticks by minutes, so force each tick's run into its own
		// millisecond rather than tightening RunsTokensSince's contract.
		time.Sleep(2 * time.Millisecond)
		since := store.LatestRunID(w)
		d.reviewPhase()
		tokens := store.RunsTokensSince(w, since)
		if tokens <= 0 {
			t.Fatalf("tick %d: expected the review spawn to report real usage.txt tokens, got %d", ticks, tokens)
		}
		gov.ChargeIdleTokens(tokens)
		ticks++
		if ticks > 10 {
			t.Fatal("window guard never tripped after 10 idle ticks — idle spend is not accumulating")
		}
	}

	// Budget 100 / 30-per-tick must take more than one idle tick to exhaust —
	// otherwise the guard could be tripping on something other than real
	// accumulation across ticks.
	if ticks < 2 {
		t.Fatalf("want the guard to trip only after multiple idle ticks accumulated (>= 2), tripped after %d", ticks)
	}
	if gov.WindowSpent() < int64(ticks)*tokensPerSpawn {
		t.Fatalf("want windowSpent to reflect every idle tick's charge (>= %d after %d ticks), got %d",
			int64(ticks)*tokensPerSpawn, ticks, gov.WindowSpent())
	}
}

// TestReviewPhaseForwardsMaxTokens is the 106 regression for acceptance
// criterion 3: the review spawn (both the Idle-branch and in-cycle callers
// share reviewPhase) must forward --max-tokens the same way the build spawn
// does in runCycle, so an idle review run is bounded per-run too.
func TestReviewPhaseForwardsMaxTokens(t *testing.T) {
	w := loopEnv(t)
	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{})
	d.cfg.perCycleTok = 500

	d.reviewPhase()

	var reviewSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" {
			reviewSpawn = c
		}
	}
	if reviewSpawn == nil {
		t.Fatal("reviewPhase did not spawn")
	}
	if !contains(reviewSpawn, "--max-tokens") || !contains(reviewSpawn, "500") {
		t.Fatalf("review spawn should forward --max-tokens 500 mirroring the build spawn, got: %v", reviewSpawn)
	}
}

// TestRecordSelfPRHoldsPushWhileBranchPending is the 114 regression: the
// record commit must not be pushed to origin while a self-PR branch this loop
// opened is still awaiting GitHub's auto-merge — a mid-cycle push there
// advances `main` out from under the queued PR, and strict branch protection
// reads that as "behind" and never merges it (issue #75). Once the branch is
// gone from origin (merged + --delete-branch, or closed), the held-back
// record catches up and pushes.
func TestRecordSelfPRHoldsPushWhileBranchPending(t *testing.T) {
	w := loopEnv(t)

	// A bare "origin" remote, with `main` and a fixer branch pushed to it —
	// standing in for the branch dacli/pr --auto opened a PR against.
	origin := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", origin).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = w.Root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	run("remote", "add", "origin", origin)
	run("add", "-A")
	run("commit", "-q", "-m", "init")
	run("push", "-q", "-u", "origin", "main")
	branch := "dacli/001-fixture"
	run("checkout", "-q", "-b", branch)
	run("commit", "-q", "--allow-empty", "-m", "fixer work")
	run("push", "-q", "-u", "origin", branch)
	run("checkout", "-q", "main")

	// An OPEN PR is what holds the record push — not the mere existence of a
	// remote branch. A leftover origin/dacli/NNN from an attempt that never
	// opened a PR is deleted by nothing, so treating the ref as proof of
	// flight held the push indefinitely and every later cycle repeated the
	// same conclusion (the wedge observed 2026-08-06).
	stubOrchestrationGH(t, func(_ string, args ...string) (string, error) {
		return `[{"state":"OPEN","autoMergeRequest":{"enabledAt":"now"}}]`, nil
	})

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{})
	d.pendingLand = []string{branch}

	d.recordSelfPR()
	shipCall := lastShipCall(fr)
	if shipCall == nil {
		t.Fatal("recordSelfPR did not run ship")
	}
	if contains(shipCall, "--push") {
		t.Fatalf("must not push while %s has an OPEN PR, got: %v", branch, shipCall)
	}
	if len(d.pendingLand) != 1 {
		t.Fatalf("a branch with an open PR must stay pending, got: %v", d.pendingLand)
	}
}

// The other half, and the actual bug: a remote branch with NO pull request
// must not hold the record push. Nothing deletes such a ref, so reading its
// existence as "in flight" wedged the loop — it never recorded anything again.
// The other half, and the actual bug: a remote branch with NO pull request
// must not hold the record push. Nothing deletes such a ref, so reading its
// existence as "in flight" wedged the loop — it stopped recording entirely
// while every cycle re-reached the same conclusion (observed 2026-08-06).
func TestRecordSelfPRDoesNotHoldForABranchWithNoPR(t *testing.T) {
	w := loopEnv(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", origin).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = w.Root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("remote", "add", "origin", origin)
	run("add", "-A")
	run("commit", "-q", "-m", "init")
	run("push", "-q", "-u", "origin", "main")
	branch := "dacli/002-stale"
	run("checkout", "-q", "-b", branch)
	run("commit", "-q", "--allow-empty", "-m", "abandoned work")
	run("push", "-q", "-u", "origin", branch)
	run("checkout", "-q", "main")

	// gh answers successfully with an empty list: the ref exists, no PR does.
	stubOrchestrationGH(t, func(_ string, _ ...string) (string, error) { return "[]", nil })

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{})
	d.pendingLand = []string{branch}

	d.recordSelfPR()
	if len(d.pendingLand) != 0 {
		t.Errorf("a branch with no PR must not stay pending, got: %v", d.pendingLand)
	}
	if call := lastShipCall(fr); call == nil || !contains(call, "--push") {
		t.Errorf("the record must push when nothing is actually in flight, got: %v", call)
	}
}

// TestLoopSyncsTrunkAfterAsyncAutoMergeBeforeRecordPush is the 159 regression:
// once task 114 lets a fixer's PR actually land under strict branch
// protection via `gh pr merge --auto`, that merge happens ASYNCHRONOUSLY —
// on GitHub's own schedule, not synchronously inside any dacli command this
// loop runs — so local main only ever falls further behind origin/main
// across cycles unless something explicitly reconciles it. Before this fix,
// nothing did: the loop would eventually try to push a record commit on top
// of a stale local main and get rejected non-fast-forward. This drives a real
// bare-origin + two-clone setup (the loop's own checkout, and a sibling
// standing in for wherever GitHub's merge landed) through d.syncTrunk() and
// proves the local checkout catches up.
func TestLoopSyncsTrunkAfterAsyncAutoMergeBeforeRecordPush(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	origin := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", origin).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}

	w := loopEnv(t)
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v (in %s): %v\n%s", args, dir, err, out)
		}
		return string(out)
	}
	run(w.Root, "remote", "add", "origin", origin)
	commitTo(t, w.Root, "seed.txt")
	run(w.Root, "push", "-q", "-u", "origin", "main")

	// A sibling clone stands in for wherever GitHub's `gh pr merge --auto`
	// actually landed the fixer's PR — a commit that reaches origin without
	// ever touching the loop's own checkout.
	sibling := filepath.Join(t.TempDir(), "sibling")
	if out, err := exec.Command("git", "clone", "-q", origin, sibling).CombinedOutput(); err != nil {
		t.Fatalf("git clone sibling: %v\n%s", err, out)
	}
	for _, a := range [][]string{{"config", "user.email", "x@x"}, {"config", "user.name", "x"}} {
		run(sibling, a...)
	}
	commitTo(t, sibling, "landed-via-gh-auto-merge.txt")
	run(sibling, "push", "-q", "origin", "main")

	d := newDriver(w, &fakeRunner{}, &Governor{})
	d.trunkBranch = d.resolveTrunkBranch()

	before := strings.TrimSpace(run(w.Root, "rev-parse", "HEAD"))
	originHead := strings.TrimSpace(run(sibling, "rev-parse", "HEAD"))
	if before == originHead {
		t.Fatal("test setup: local checkout must start behind the async merge, not already caught up")
	}

	d.syncTrunk()

	after := strings.TrimSpace(run(w.Root, "rev-parse", "HEAD"))
	if after != originHead {
		t.Fatalf("syncTrunk did not fast-forward local main to the async auto-merge: got %s want %s", after, originHead)
	}

	// With local main reconciled, the record commit ship makes next sits ON
	// TOP of the merged state — its own push is a clean fast-forward, never a
	// non-fast-forward rejection.
	commitTo(t, w.Root, "record.txt")
	if out, err := gitx.Push(w.Root, "main"); err != nil {
		t.Fatalf("record push after syncTrunk should be a clean fast-forward, got: %v (%s)", err, out)
	}
}

// TestLoopFullArcDefersAcceptThenClosesOnlyAfterMergeConfirmed is the 115
// end-to-end acceptance test: it drives the exact sequence loop() itself
// would — runCycle builds and defers a task's accept, the task is excluded
// from the next ready frontier while its PR is in flight, and only once a
// later reconciliation confirms the merge does the record actually close.
// Before this fix, accept --force fired the moment the PR opened; this proves
// it now waits for trunk-backed confirmation, per issue #74.
func TestLoopFullArcDefersAcceptThenClosesOnlyAfterMergeConfirmed(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt")
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateTask(w, "a-root", "p", "Untouched sibling", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	r := &spawnOutcomeRunner{w: w}
	d := newDriver(w, r, &Governor{})
	d.cfg.width = 1

	// Cycle N: the task builds; its accept must be deferred, not fired.
	d.runCycle([]*store.Task{task})
	for _, c := range r.calls {
		if len(c) > 1 && c[0] == "accept" {
			t.Fatalf("accept must not fire in the build cycle itself: %v", c)
		}
	}

	// Between cycle N and N+1 (what loop() does every iteration): compute the
	// ready frontier and exclude anything still awaiting merge confirmation.
	ready, err := readyTasks(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	ready = excludePending(ready, d.pendingAccept)
	for _, t2 := range ready {
		if t2.Seq == task.Seq {
			t.Fatalf("task %03d must be excluded from the ready frontier while its PR is in flight", task.Seq)
		}
	}
	foundOther := false
	for _, t2 := range ready {
		if t2.Seq == other.Seq {
			foundOther = true
		}
	}
	if !foundOther {
		t.Fatalf("the untouched sibling task must still be ready, got: %v", ready)
	}

	// Cycle N+1: gh now confirms the PR merged — reconciliation closes it.
	stubOrchestrationGH(t, func(dir string, args ...string) (string, error) {
		return `[{"state":"MERGED"}]`, nil
	})
	d.reconcilePendingAccepts()

	done, err := store.ListTasks(w, "p", model.StatusDone)
	if err != nil {
		t.Fatal(err)
	}
	foundDone := false
	for _, tk := range done {
		if tk.Seq == task.Seq {
			foundDone = true
		}
	}
	if !foundDone {
		t.Fatalf("task %03d should be closed once its PR merge is confirmed", task.Seq)
	}
	if len(d.pendingAccept) != 0 {
		t.Fatalf("pendingAccept should be empty after a confirmed merge, got: %v", d.pendingAccept)
	}
}

// --- stage-aware loop (dacli 189) ---

// twoStageManifest is a minimal phase-gated template for the loop tests: one
// pre-implementation stage that admits no implementer (the shape `dacli init
// --template product` produces, which used to deadlock the loop) followed by an
// implementation stage that does. Its gates are the cheapest real predicates —
// a filled Goal, then all musts done — so a test controls exactly which gate is
// open without touching the shipped manifests.
const twoStageManifest = `---
name: looptest
summary: two phase-gated stages for the loop tests
cost: test fixture
---
# looptest

## stage: discovery
cone: definition
phase: discovery
allow: researcher, reviewer
- project_sections: Goal

## stage: build
cone: design
phase: implementation
allow: implementer, reviewer
- tasks: musts_done
`

// attachTemplate vendors manifest into the workspace's template dir and binds
// it to project p — the same path `dacli template add` + `project add
// --template` take, so the test exercises real frontmatter, not a stub.
func attachTemplate(t *testing.T, w *workspace.Workspace, name, manifest string) {
	t.Helper()
	if err := os.MkdirAll(w.TemplatesDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.TemplatesDir(), name+".md"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := gates.Attach(w, "p", name); err != nil {
		t.Fatalf("attach template %s: %v", name, err)
	}
}

// fillGoal writes a Goal that satisfies the gates package's filled-not-present
// rule (long enough, no placeholder, unambiguous), opening the discovery gate.
func fillGoal(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	p, err := store.LoadProject(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.SetSection("Goal", "Ship a command line tool that converts CSV files into JSON files.\n")
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
}

func seedRoles(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	for _, r := range []team.Role{
		{Name: "fixer", Kind: "implementer", Summary: "writes code"},
		{Name: "scout", Kind: "researcher", Summary: "investigates"},
	} {
		if err := store.CreateRole(w, "a-root", r); err != nil {
			t.Fatal(err)
		}
	}
}

func buildSpawnCall(fr *fakeRunner) []string {
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "--detach") {
			return c
		}
	}
	return nil
}

// buildSpawnCalls returns every BUILD-phase spawn call in the cycle (one per
// batched task), as opposed to buildSpawnCall's first-match.
func buildSpawnCalls(fr *fakeRunner) [][]string {
	var out [][]string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "--detach") {
			out = append(out, c)
		}
	}
	return out
}

// spawnRoleForTask returns the --role value of the build spawn issued for
// task ref, or "" if none was found.
func spawnRoleForTask(calls [][]string, ref string) string {
	for _, c := range calls {
		hasRef, role := false, ""
		for i, a := range c {
			if a == "--task" && i+1 < len(c) {
				hasRef = c[i+1] == ref
			}
			if a == "--role" && i+1 < len(c) {
				role = c[i+1]
			}
		}
		if hasRef {
			return role
		}
	}
	return ""
}

// TestLoopAdvancesStageWhenEveryGateCheckPasses is the 189 regression: the loop
// used to import `gates` nowhere, so a phase-gated project sat in its first
// stage forever — every implementer spawn refused by phaseGate ("advance the
// stage first"), and nothing in an autonomous run ever advances a stage. Here
// the discovery gate's one check is satisfied, so the loop must advance the
// project into the implementation phase itself and then build with the
// implementer. The build gate stays SHUT (an open `must` task) so the test also
// proves the loop stops at a closed gate rather than running the manifest out.
func TestLoopAdvancesStageWhenEveryGateCheckPasses(t *testing.T) {
	w := loopEnv(t)
	fillGoal(t, w)
	attachTemplate(t, w, "looptest", twoStageManifest)
	seedRoles(t, w)
	if _, err := store.CreateTask(w, "a-root", "p", "Convert the CSV reader", store.TaskOpts{
		Priority: "must", Accept: []string{"a"},
	}); err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	if got := d.templateStage(); got != "build" {
		t.Fatalf("loop must advance past the passing discovery gate and stop at the shut build gate, stage is %q", got)
	}
	ph, err := gates.PhaseFor(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if ph.Name != "implementation" {
		t.Fatalf("advancing the stage must move the project into the implementation phase, got %q", ph.Name)
	}
	spawn := buildSpawnCall(fr)
	if spawn == nil {
		t.Fatalf("no build spawn after the stage advanced, calls: %v", fr.calls)
	}
	if !contains(spawn, "fixer") {
		t.Fatalf("the implementation phase admits the implementer — build spawn should use it, got: %v", spawn)
	}
}

// TestLoopBuildsWithPhaseAppropriateRoleWhenImplementerIsBarred is the other
// half of 189: while a gate is genuinely shut (the Goal here is unfilled, so
// discovery holds), the loop must not keep firing spawns the phase will refuse.
// The discovery phase admits researcher and reviewer kinds, so the loop builds
// with the roster's researcher instead of the configured implementer.
func TestLoopBuildsWithPhaseAppropriateRoleWhenImplementerIsBarred(t *testing.T) {
	w := loopEnv(t) // Goal left as loopEnv's one-character stub: the gate stays shut
	attachTemplate(t, w, "looptest", twoStageManifest)
	seedRoles(t, w)
	if _, err := store.CreateTask(w, "a-root", "p", "Some work", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	if got := d.templateStage(); got != "discovery" {
		t.Fatalf("an unmet gate must hold the project at discovery, got %q", got)
	}
	spawn := buildSpawnCall(fr)
	if spawn == nil {
		t.Fatalf("no build spawn at all, calls: %v", fr.calls)
	}
	if contains(spawn, "fixer") {
		t.Fatalf("discovery admits only researcher/reviewer — spawning the implementer is a guaranteed refusal: %v", spawn)
	}
	if !contains(spawn, "scout") {
		t.Fatalf("build spawn should use the roster's researcher in the discovery phase, got: %v", spawn)
	}
}

// TestUntemplatedProjectIsNeverStageGated is the no-regression guard for 189:
// the overwhelmingly common solo project has no template, and every stage-aware
// path added for the gated case must be inert for it — the configured
// implementer is used verbatim and nothing is written to the log.
func TestUntemplatedProjectIsNeverStageGated(t *testing.T) {
	w := loopEnv(t) // no template attached
	seedRoles(t, w) // a researcher on the roster must NOT tempt the picker

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{})
	out := d.ctx.Stdout.(*bytes.Buffer)

	if got := d.templateStage(); got != "" {
		t.Fatalf("an untemplated project has no stage, got %q", got)
	}
	d.advanceStages()
	if got := d.buildRole(); got != "fixer" {
		t.Fatalf("an untemplated project must build with the configured implementer, got %q", got)
	}
	if out.Len() != 0 {
		t.Fatalf("stage handling must be silent for an untemplated project, logged: %q", out.String())
	}
}

// TestLoopRoutesEachTaskToCheapestCapableRoleByTe is the 233 regression: the
// loop used to resolve ONE role per cycle (cfg.implRole, or its phase-gated
// substitute) and spawn every batched task with it, ignoring that
// team.CheapestCapable already exists to pick the cheapest role whose capacity
// covers a task's own Te (dacli 230, 231). A roster with a capped cheap role
// and an uncapped expensive one must route small and large work differently,
// even within the same cycle/batch.
func TestLoopRoutesEachTaskToCheapestCapableRoleByTe(t *testing.T) {
	w := loopEnv(t)
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "fixer", Kind: "implementer", Model: "opus", Summary: "writes code",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "junior-fixer", Kind: "implementer", Model: "haiku", MaxPoints: 3, Summary: "writes small fixes",
	}); err != nil {
		t.Fatal(err)
	}

	small, err := store.CreateTask(w, "a-root", "p", "Small fix", store.TaskOpts{Accept: []string{"a"}, Estimate: "1,1,1"})
	if err != nil {
		t.Fatal(err)
	}
	large, err := store.CreateTask(w, "a-root", "p", "Big rewrite", store.TaskOpts{Accept: []string{"a"}, Estimate: "8,10,12"})
	if err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	calls := buildSpawnCalls(fr)
	smallRole := spawnRoleForTask(calls, fmt.Sprintf("%03d", small.Seq))
	largeRole := spawnRoleForTask(calls, fmt.Sprintf("%03d", large.Seq))
	if smallRole != "junior-fixer" {
		t.Errorf("small task (Te 1, within junior-fixer's cap of 3) routed to %q, want junior-fixer — the loop must route by team.CheapestCapable per task, not always cfg.implRole", smallRole)
	}
	if largeRole != "fixer" {
		t.Errorf("large task (Te 10, above junior-fixer's cap) routed to %q, want fixer (the only role whose capacity covers it)", largeRole)
	}
}

// --- greenfield spec decomposition (dacli 190) ---

func anchorTask(t *testing.T, w *workspace.Workspace, ref string) *store.Task {
	t.Helper()
	task, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

// TestGreenfieldProjectGetsSpecDecompositionAnchor is the 190 regression: with a
// Goal and zero tasks over an empty repo, the evidence-based anchor asks for
// work "grounded in evidence (a failing test, a reviewer finding, a real
// defect)" and forbids inventing any — so nothing is ever filed and the loop
// idles forever. Such a project must get the spec-decomposition anchor instead,
// which is licensed to derive the backlog from the project's stated intent.
func TestGreenfieldProjectGetsSpecDecompositionAnchor(t *testing.T) {
	w := loopEnv(t) // a project with a Goal, no tasks, no tracked code
	d := newDriver(w, &fakeRunner{}, &Governor{})

	ref, err := d.ensureImproveTask()
	if err != nil {
		t.Fatal(err)
	}
	task := anchorTask(t, w, ref)
	if task.Title != specAnchorTitle {
		t.Fatalf("a greenfield project must get the spec-decomposition anchor, got %q", task.Title)
	}
	if !task.IsLoopAnchor() {
		t.Fatal("the spec anchor must still read as a loop anchor, or the loop would hand it to a builder as ordinary work")
	}
	ctxSec, ok := task.Doc.Section("Context")
	if !ok {
		t.Fatal("spec anchor has no Context section")
	}
	for _, want := range []string{"Goal", "--depends-on", "--parent", "task add"} {
		if !strings.Contains(ctxSec.Content, want) {
			t.Fatalf("spec anchor charter must tell the agent to file dependency-ordered work (missing %q):\n%s", want, ctxSec.Content)
		}
	}

	// The anchor is a prompt, not work: it must stay out of the ready frontier.
	ready, err := readyTasks(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 0 {
		t.Fatalf("the anchor must never enter the ready frontier, got: %v", ready)
	}
}

// TestExistingCodebaseKeepsEvidenceBasedAnchor is 190's no-regression half: a
// repo that already has code has evidence to survey even when its backlog is
// momentarily empty, so it must keep the original evidence-based charter — the
// spec anchor's licence to invent work is only for a project with nothing to
// stand on.
func TestExistingCodebaseKeepsEvidenceBasedAnchor(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "main.go") // real source in the repo, empty backlog
	d := newDriver(w, &fakeRunner{}, &Governor{})

	ref, err := d.ensureImproveTask()
	if err != nil {
		t.Fatal(err)
	}
	if got := anchorTask(t, w, ref).Title; got != evidenceAnchorTitle {
		t.Fatalf("a project with code must keep the evidence-based anchor, got %q", got)
	}

	// One filed task is also enough on its own: somebody decomposed the goal
	// already, so re-decomposing it would duplicate their work.
	w2 := loopEnv(t)
	if _, err := store.CreateTask(w2, "a-root", "p", "Already decomposed", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	d2 := newDriver(w2, &fakeRunner{}, &Governor{})
	ref2, err := d2.ensureImproveTask()
	if err != nil {
		t.Fatal(err)
	}
	if got := anchorTask(t, w2, ref2).Title; got != evidenceAnchorTitle {
		t.Fatalf("a project with a real backlog must keep the evidence-based anchor, got %q", got)
	}
}

// TestGreenfieldLoopReviewsAgainstSpecAnchorInsteadOfIdling drives the whole
// 190 arc through loop(): an empty backlog reaches the Idle branch, whose review
// phase must spawn against the spec-decomposition anchor — the change that lets
// a greenfield project file its first work instead of printing "idling rather
// than inventing work" forever.
func TestGreenfieldLoopReviewsAgainstSpecAnchorInsteadOfIdling(t *testing.T) {
	w := loopEnv(t)
	fr := &filingRunner{w: w, reviewRole: "go-auditor"}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3, Idle: time.Millisecond})
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	var reviewSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "go-auditor") {
			reviewSpawn = c
			break
		}
	}
	if reviewSpawn == nil {
		t.Fatalf("the idle branch never ran a review spawn, calls: %v", fr.calls)
	}
	if got := anchorTask(t, w, argAfter(reviewSpawn, "--task")).Title; got != specAnchorTitle {
		t.Fatalf("the greenfield review must run against the spec-decomposition anchor, got %q", got)
	}
	if fr.filedRef == "" {
		t.Fatal("the review phase filed nothing — the greenfield project would idle forever")
	}
}

func lastShipCall(fr *fakeRunner) []string {
	var out []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "ship" {
			out = c
		}
	}
	return out
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// refusingReviewRunner simulates a spawn refusal when the review role tries to spawn.
// It records both the spawn attempt and the refusal text.
type refusingReviewRunner struct {
	fakeRunner
	reviewRole      string
	refusalMessage  string
	reviewSpawnSeen bool
}

func (r *refusingReviewRunner) run(label string, args ...string) (string, error) {
	r.fakeRunner.run(label, args...)
	if len(args) > 0 && args[0] == "spawn" && contains(args, r.reviewRole) {
		r.reviewSpawnSeen = true
		return r.refusalMessage, fmt.Errorf("exit status 3")
	}
	return "", nil
}

// TestReviewPhaseSurfacesSpawnRefusal checks that when the review spawn fails
// due to capacity constraints (or other refusals), the refusal is reported
// rather than silently swallowed. This ensures an idle loop that generates no
// work due to a capped role says why.
func TestReviewPhaseSurfacesSpawnRefusal(t *testing.T) {
	w := loopEnv(t)
	refusalText := "spawn refused: capacity exceeded for role go-auditor"
	rr := &refusingReviewRunner{reviewRole: "go-auditor", refusalMessage: refusalText}
	d := newDriver(w, rr, &Governor{})

	d.reviewPhase()

	// Verify the review spawn was attempted
	if !rr.reviewSpawnSeen {
		t.Fatal("review phase did not attempt to spawn the reviewer")
	}

	// Verify the refusal was logged in the context output
	logOutput := d.ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(logOutput, "refused") || !strings.Contains(logOutput, refusalText) {
		t.Fatalf("review phase did not report the spawn refusal; got: %s", logOutput)
	}
}

// timeoutReviewRunner simulates a review spawn that times out.
type timeoutReviewRunner struct {
	fakeRunner
	reviewRole string
	timeoutMsg string
}

func (r *timeoutReviewRunner) run(label string, args ...string) (string, error) {
	r.fakeRunner.run(label, args...)
	if len(args) > 0 && args[0] == "spawn" && contains(args, r.reviewRole) {
		// Simulate the output from spawn when it times out:
		// First the stderr banner from spawn, then the actual error message
		return r.timeoutMsg, fmt.Errorf("child a-auditor-123: stalled (see /some/run/dir)")
	}
	return "", nil
}

// TestReviewPhaseReportsTimeoutDistinctlyFromRefusal verifies that when the review
// spawn times out, the log reports it as a timeout with elapsed time and limit, not
// as a policy refusal. This ensures the operator can distinguish a timeout kill from
// a policy refusal in the output alone.
func TestReviewPhaseReportsTimeoutDistinctlyFromRefusal(t *testing.T) {
	w := loopEnv(t)
	timeoutMsg := "spawning a-auditor-123 on go-runtime for 001\nrun 1234abcd\n"
	tr := &timeoutReviewRunner{reviewRole: "go-auditor", timeoutMsg: timeoutMsg}
	d := newDriver(w, tr, &Governor{})
	d.cfg.perCycleTok = 60000 // Set a timeout so we can verify it's mentioned

	d.reviewPhase()

	// Verify the review spawn was attempted
	if !contains(tr.firstArgs(), "spawn") {
		t.Fatal("review phase did not attempt to spawn the reviewer")
	}

	// Verify the timeout was logged with the word "timeout" and NOT with "refused"
	logOutput := d.ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(logOutput, "refused") {
		t.Fatalf("review phase reported 'refused' for a timeout; got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "timeout") && !strings.Contains(logOutput, "timed out") && !strings.Contains(logOutput, "stalled") {
		t.Fatalf("review phase did not report timeout; got: %s", logOutput)
	}

	// Verify the spawn success banner is NOT printed as an error
	if strings.Contains(logOutput, "spawning a-auditor-123") {
		t.Fatalf("review phase printed the spawn success banner as part of error; got: %s", logOutput)
	}
}

// TestReviewAnchorHasEstimate checks that the standing review anchor carries
// an estimate so a capacity-capped review role can accept it during spawning.
func TestReviewAnchorHasEstimate(t *testing.T) {
	w := loopEnv(t)
	d := newDriver(w, &fakeRunner{}, &Governor{})

	// Ensure the review anchor task exists
	ref, err := d.ensureImproveTask()
	if err != nil {
		t.Fatalf("ensureImproveTask failed: %v", err)
	}

	// Load the task and verify it has an estimate
	refInt := 0
	fmt.Sscanf(ref, "%d", &refInt)
	tasks, err := store.ListTasks(w, "p", "")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	var anchor *store.Task
	for _, tk := range tasks {
		if tk.Seq == refInt {
			anchor = tk
			break
		}
	}
	if anchor == nil {
		t.Fatalf("could not find anchor task with ref %s", ref)
	}

	// Verify the anchor has an estimate
	tp, ok := anchor.Estimate()
	if !ok {
		t.Fatal("review anchor task has no estimate — capacity-capped roles cannot schedule it")
	}
	// The estimate should be "1,2,3" as set in ensureImproveTask
	if tp.Optimistic != 1 || tp.Probable != 2 || tp.Pessimistic != 3 {
		t.Fatalf("unexpected estimate; got %v", tp)
	}
}

// The verbatim failure from run 01KZPAKYFN, which is what motivated dacli 333:
// a go-auditor ran for five minutes, was killed on a timeout, and the loop
// printed the spawn's SUCCESS banner as the failure message. The earlier test
// uses a simplified banner; this one uses the real one, because the real
// format is what the parsing has to survive.
func TestReviewFailureNeverReportsTheSuccessBannerAsTheCause(t *testing.T) {
	realBanner := "spawning a-go-auditor-yn0a9b on cc for 303-continuous-improvement-file-the-single-highest-value-evidence-based-change (run 01KZPAKYFN)\n"
	got := reviewFailure(realBanner, fmt.Errorf("child a-go-auditor-yn0a9b: stalled"))

	if strings.Contains(got, "spawning a-go-auditor-yn0a9b") {
		t.Errorf("the spawn banner was reported as the failure cause: %s", got)
	}
	if !strings.Contains(got, "stalled") {
		t.Errorf("the actual error must be reported: %s", got)
	}
	// The run record is the thing that knows the outcome, so point at it by id.
	if !strings.Contains(got, "01KZPAKYFN") {
		t.Errorf("the run id must be named so the operator can look it up: %s", got)
	}
	// A spawn that STARTED must not be described as refused — that is the
	// distinction the whole task is about.
	if strings.Contains(got, "refused") {
		t.Errorf("a started-then-killed spawn is not a refusal: %s", got)
	}

	// And the mirror: a spawn the gate never let through has no banner, so the
	// output IS the refusal and must be quoted.
	ref := reviewFailure("dacli: role go-auditor takes only estimated tasks (max 8 points)\n", fmt.Errorf("exit status 3"))
	if !strings.Contains(ref, "refused") || !strings.Contains(ref, "only estimated tasks") {
		t.Errorf("a real refusal must be quoted verbatim: %s", ref)
	}

	// No output at all still says something actionable rather than nothing.
	if bare := reviewFailure("", fmt.Errorf("boom")); !strings.Contains(bare, "boom") {
		t.Errorf("with no output the error itself must be reported: %s", bare)
	}
}
