package orchestration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// TestRunCycleSyncsBeforeLandNotAfter is the 299 acceptance case for "a
// cycle applies proposals from read-only agents before it judges whether the
// cycle produced anything": SYNC must run right after the wave finalizes
// (WAIT) and before the LAND step (--pr's self-PR bookkeeping, or --no-pr's
// ship integrate) — never after. The loop's trunk-marker judgment (loop())
// runs after runCycle returns, so ordering sync ahead of LAND is what
// guarantees a read-only agent's proposal is folded in before that judgment.
func TestRunCycleSyncsBeforeLandNotAfter(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{})
	d.cfg.width = 1
	d.cfg.pr = false // land model exercised: "ship"
	d.runCycle([]*store.Task{task})

	order := fr.firstArgs()
	waitAt, syncAt, shipAt := -1, -1, -1
	for i, a := range order {
		switch a {
		case "wait":
			if waitAt == -1 {
				waitAt = i
			}
		case "sync":
			if syncAt == -1 {
				syncAt = i
			}
		case "ship":
			if shipAt == -1 {
				shipAt = i
			}
		}
	}
	if waitAt == -1 || syncAt == -1 || shipAt == -1 {
		t.Fatalf("expected wait, sync, and ship to all run, got order: %v", order)
	}
	if !(waitAt < syncAt && syncAt < shipAt) {
		t.Fatalf("want wait < sync < ship, got wait=%d sync=%d ship=%d (order: %v)", waitAt, syncAt, shipAt, order)
	}
}

// blockingSyncRunner stands in for a real spawn+sync round trip: its "spawn"
// gives the named task a real branch+commit (like spawnOutcomeRunner), and
// its "sync" call moves the task straight to Blocked — the durable effect
// eventlog.Sync has on a task whose read-only build agent could only
// PROPOSE a block (`dacli task block`, planning.go cmdTaskBlock) because a
// spawned agent's default grant cannot mutate a task it does not own. This
// lets a test prove classifyBatch reads the POST-sync status.
type blockingSyncRunner struct {
	fakeRunner
	w        *workspace.Workspace
	blockRef string
}

func (r *blockingSyncRunner) run(label string, args ...string) (string, error) {
	_, _ = r.fakeRunner.run(label, args...)
	switch {
	case len(args) > 0 && args[0] == "spawn":
		ref := argAfter(args, "--task")
		t, err := store.FindTask(r.w, ref)
		if err != nil {
			return "", err
		}
		if err := branchWithCommit(r.w.Root, taskBranch(t)); err != nil {
			return "", err
		}
		return "", nil
	case len(args) > 0 && args[0] == "sync":
		t, err := store.FindTask(r.w, r.blockRef)
		if err != nil {
			return "", err
		}
		return "", store.MoveTask(r.w, t, model.StatusBlocked)
	}
	return "", nil
}

// TestClassifyBatchReportsBlockedAfterSyncApplication is the AC3+AC4
// combination: the rollup's Blocked bucket must reflect the status SYNC
// left the task in, not the status it had at spawn time — proving sync ran,
// and was applied, before the cycle's own judgment of what happened.
func TestClassifyBatchReportsBlockedAfterSyncApplication(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt")
	task, err := store.CreateTask(w, "a-root", "p", "Task a read-only agent blocks on a question", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	r := &blockingSyncRunner{w: w, blockRef: fmt.Sprintf("%03d", task.Seq)}
	d := newDriver(w, r, &Governor{})
	d.cfg.width = 1

	_, rollup := d.runCycle([]*store.Task{task})

	if rollup.Blocked != 1 {
		t.Fatalf("want rollup.Blocked=1 once sync applies the block, got %+v", rollup)
	}
	if rollup.Landed != 0 || rollup.ProducedNothing != 0 {
		t.Fatalf("a blocked task must not double-count as landed or produced-nothing, got %+v", rollup)
	}
}

// TestClassifyBatchCountsRefusedSpawnAsProducedNothing is the plain "no work
// at all" bucket: a task whose spawn never even launched must not be
// confused with one that built and is merely still in flight.
func TestClassifyBatchCountsRefusedSpawnAsProducedNothing(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt")
	refused, err := store.CreateTask(w, "a-root", "p", "Task whose spawn is refused", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := store.CreateTask(w, "a-root", "p", "Task that builds fine", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	r := &spawnOutcomeRunner{w: w, refusedRef: refused.ID}
	d := newDriver(w, r, &Governor{})
	d.cfg.width = 2

	_, rollup := d.runCycle([]*store.Task{refused, ok})

	if rollup.ProducedNothing != 1 {
		t.Fatalf("want exactly 1 produced-nothing (the refused spawn), got %+v", rollup)
	}
	if rollup.Stalled != 1 {
		t.Fatalf("want the successfully built (but --pr-pending) sibling counted stalled, got %+v", rollup)
	}
}

// TestClassifyBatchCountsLocalShipIntegrationAsLanded is the --no-pr LAND
// model: once ship's own accept+integrate closes the task within the same
// cycle, the rollup must report it landed, not merely stalled.
func TestClassifyBatchCountsLocalShipIntegrationAsLanded(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt")
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	// shipClosesTaskRunner simulates the one real effect `dacli ship` has on
	// the world for this test: closing the task it integrated.
	r := &shipClosesTaskRunner{w: w, ref: fmt.Sprintf("%03d", task.Seq)}
	d := newDriver(w, r, &Governor{})
	d.cfg.width = 1
	d.cfg.pr = false

	_, rollup := d.runCycle([]*store.Task{task})

	if rollup.Landed != 1 {
		t.Fatalf("want the ship-integrated task counted landed, got %+v", rollup)
	}
}

type shipClosesTaskRunner struct {
	fakeRunner
	w   *workspace.Workspace
	ref string
}

func (r *shipClosesTaskRunner) run(label string, args ...string) (string, error) {
	_, _ = r.fakeRunner.run(label, args...)
	switch {
	case len(args) > 0 && args[0] == "spawn":
		t, err := store.FindTask(r.w, argAfter(args, "--task"))
		if err != nil {
			return "", err
		}
		return "", branchWithCommit(r.w.Root, taskBranch(t))
	case len(args) > 0 && args[0] == "ship":
		t, err := store.FindTask(r.w, r.ref)
		if err != nil {
			return "", err
		}
		return "", store.MoveTask(r.w, t, model.StatusDone)
	}
	return "", nil
}

// TestReconcilePendingAcceptsRollupCountsMergeAsLanded extends the 115
// reconcile behavior with its rollup contribution (dacli 299): a confirmed
// merge must roll up as Landed.
func TestReconcilePendingAcceptsRollupCountsMergeAsLanded(t *testing.T) {
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
	d.pendingAccept = []pendingAccept{{Seq: task.Seq, Branch: taskBranch(task)}}

	got := d.reconcilePendingAccepts()
	want := cycleRollup{Landed: 1}
	if got != want {
		t.Fatalf("want rollup %+v for a confirmed merge, got %+v", want, got)
	}
}

// TestReconcilePendingAcceptsRollupCountsOrphanAsProducedNothing mirrors the
// above for a PR that closed unmerged: no work reached trunk, so it rolls up
// as produced-nothing, not stalled (stalled means still in flight; an
// orphaned PR is a concluded outcome).
func TestReconcilePendingAcceptsRollupCountsOrphanAsProducedNothing(t *testing.T) {
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

	got := d.reconcilePendingAccepts()
	want := cycleRollup{ProducedNothing: 1}
	if got != want {
		t.Fatalf("want rollup %+v for an orphaned PR, got %+v", want, got)
	}
}

// TestReconcilePendingAcceptsRollupCountsStrandedAsStalled mirrors task 290's
// stranded case: a PR neither merged nor abandoned rolls up as stalled —
// still in flight, needing attention, but not yet a concluded outcome.
func TestReconcilePendingAcceptsRollupCountsStrandedAsStalled(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	stubOrchestrationGH(t, func(dir string, args ...string) (string, error) {
		return `[{"state":"OPEN"}]`, nil
	})
	r := &fakeRunner{}
	d := newDriver(w, r, &Governor{})
	d.pendingAccept = []pendingAccept{{Seq: task.Seq, Branch: taskBranch(task)}}

	got := d.reconcilePendingAccepts()
	want := cycleRollup{Stalled: 1}
	if got != want {
		t.Fatalf("want rollup %+v for a stranded PR, got %+v", want, got)
	}
}

// TestLoopPersistsRollupForStatusToRead is the AC4 end-to-end case: `dacli
// loop status` must surface the rollup from the SAME persisted snapshot
// `loop status` already reads for cycle/trunk-marker/backlog, round-tripping
// through readLoopState/writeLoopState.
func TestLoopPersistsRollupForStatusToRead(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt")
	task, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}

	r := &spawnOutcomeRunner{w: w}
	gov := &Governor{MaxCycles: 1, NoProgressHalt: 3}
	d := newDriver(w, r, gov)
	d.cfg.width = 1
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	st, err := readLoopState(w, "p")
	if err != nil {
		t.Fatalf("expected a persisted loop state, got error: %v", err)
	}
	if st.Rollup.Stalled != 1 {
		t.Fatalf("want the built-and-pending task rolled up as stalled in the persisted snapshot, got %+v", st.Rollup)
	}
	_ = task
}

// A count tells an operator something went wrong and leaves them to open six
// transcripts to find out what to do — the work the rollup exists to replace.
// Every non-landing outcome must name the command that answers "and now what?".
func TestRollupNamesARecoveryForEachNonLandingOutcome(t *testing.T) {
	r := cycleRollup{Landed: 1, ProducedNothing: 2, Stalled: 3, Blocked: 1}
	lines := r.Recovery()
	if len(lines) != 3 {
		t.Fatalf("want one recovery line per non-landing outcome, got %d: %v", len(lines), lines)
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"dacli runs list", "dacli pr status", "dacli threads"} {
		if !strings.Contains(joined, want) {
			t.Errorf("recovery missing an actionable command %q:\n%s", want, joined)
		}
	}
}

// A clean cycle says nothing: advice nobody needs is noise, and noise is what
// makes an operator stop reading the useful lines.
func TestRollupIsSilentWhenEverythingLanded(t *testing.T) {
	if lines := (cycleRollup{Landed: 4}).Recovery(); len(lines) != 0 {
		t.Errorf("a clean cycle must produce no recovery advice, got %v", lines)
	}
}

// Only the outcomes that actually occurred are reported.
func TestRollupReportsOnlyWhatHappened(t *testing.T) {
	lines := (cycleRollup{Landed: 1, Blocked: 2}).Recovery()
	if len(lines) != 1 || !strings.Contains(lines[0], "blocked") {
		t.Errorf("want only the blocked recovery, got %v", lines)
	}
}

// An unsized task silently loses BOTH orderings the loop appears to be using:
// capacity routing skips CheapestCapable, and the wave reverts to MoSCoW while
// still printing as if the critical path were in play. The loop must not leave
// that unsaid.
func TestSizeUnestimatedSaysWhatDegradesWhenItCannotSize(t *testing.T) {
	w := loopWS(t)
	task, err := store.CreateTask(w, "a-root", "core", "unsized work", store.TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}

	var logged strings.Builder
	d := &driver{
		w: w, cfg: loopCfg{project: "core"},
		ctx: &clikit.Ctx{Stdout: &logged, Stderr: &logged},
		run: &recordingRunner{},
	}
	d.sizeUnestimated([]*store.Task{task})

	got := logged.String()
	// With no estimator in the roster there is nothing honest to do but name
	// the degradation and how to fix it.
	for _, want := range []string{"no estimate", "estimator", "dacli task estimate"} {
		if !strings.Contains(got, want) {
			t.Errorf("the warning must name %q so the operator can act:\n%s", want, got)
		}
	}
}

// A fully-sized batch must be silent: a message that fires every cycle when
// nothing is wrong is a message that stops being read.
func TestSizeUnestimatedIsSilentOnASizedBatch(t *testing.T) {
	w := loopWS(t)
	task, err := store.CreateTask(w, "a-root", "core", "sized work",
		store.TaskOpts{Accept: []string{"x"}, Estimate: "1,2,4"})
	if err != nil {
		t.Fatal(err)
	}
	var logged strings.Builder
	d := &driver{
		w: w, cfg: loopCfg{project: "core"},
		ctx: &clikit.Ctx{Stdout: &logged, Stderr: &logged},
		run: &recordingRunner{},
	}
	d.sizeUnestimated([]*store.Task{task})
	if s := strings.TrimSpace(logged.String()); s != "" {
		t.Errorf("a sized batch must produce no output, got:\n%s", s)
	}
}

// recordingRunner satisfies the runner interface without launching anything.
type recordingRunner struct{ calls [][]string }

func (r *recordingRunner) run(phase string, args ...string) (string, error) {
	r.calls = append(r.calls, append([]string{phase}, args...))
	return "", nil
}

func loopWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "l")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

// The loop's BUILD ranking and `dacli next` are documented to agree. They did
// not: cmdNext excludes the standing review anchor from the CPM scheduling
// set, criticalPathSlack did not — and the anchor is created UNSIZED and never
// sized (sizeUnestimated only sizes the wave batch, which readiness filters
// anchors out of). So every steady-state cycle failed t.Estimate() on the
// anchor, dropped to MoSCoW+seq, and showed the operator critical-path order
// for a build that did not use it.
func TestCriticalPathIgnoresTheUnsizedLoopAnchor(t *testing.T) {
	w := loopWS(t)
	if _, err := store.CreateTask(w, "a-root", "core", "real sized work",
		store.TaskOpts{Accept: []string{"x"}, Estimate: "1,2,3"}); err != nil {
		t.Fatal(err)
	}
	// The anchor: unsized, open, and carrying the marker IsLoopAnchor reads.
	if _, err := store.CreateTask(w, "a-root", "core",
		store.ContinuousImprovementMarker+": file the next change",
		store.TaskOpts{Accept: []string{"filed"}}); err != nil {
		t.Fatal(err)
	}

	if _, haveCPM := criticalPathSlack(w, "core"); !haveCPM {
		t.Error("an unsized ANCHOR must not disable critical-path ordering — every real task here is sized, and this is the steady state of every cycle")
	}

	// A genuinely unsized REAL task still disables it: that degradation is
	// correct and is what sizeUnestimated exists to prevent.
	if _, err := store.CreateTask(w, "a-root", "core", "unsized real work",
		store.TaskOpts{Accept: []string{"y"}}); err != nil {
		t.Fatal(err)
	}
	if _, haveCPM := criticalPathSlack(w, "core"); haveCPM {
		t.Error("an unsized REAL task must still drop CPM — that signal is the point")
	}
}
