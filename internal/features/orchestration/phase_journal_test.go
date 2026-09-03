package orchestration

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestPhaseJournalRoundTripsEveryCrashBoundary(t *testing.T) {
	w := loopEnv(t)
	journal := cyclePhaseJournal{Project: "p", Cycle: 12}
	for i, phase := range phaseOrder {
		journal.Tasks = append(journal.Tasks, taskPhaseCheckpoint{
			TaskID: "p/task-" + string(rune('a'+i)), Sequence: i + 1, Branch: "dacli/task", Phase: phase,
			UpdatedAt: time.Unix(int64(i+1), 0).UTC(),
		})
	}
	if err := writePhaseJournal(w, journal); err != nil {
		t.Fatal(err)
	}
	got, err := readPhaseJournal(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != cyclePhaseSchema || got.Version != 1 || got.Cycle != 12 || !slices.Equal(got.Tasks, journal.Tasks) {
		t.Fatalf("phase journal did not round-trip: %+v", got)
	}
}

func TestRestartAfterSpawnWaitAndVerificationDoesNotSpawnDuplicate(t *testing.T) {
	for _, phase := range []cyclePhase{phaseSpawned, phaseWaited, phaseCommitted, phaseVerified} {
		t.Run(string(phase), func(t *testing.T) {
			w := loopEnv(t)
			commitTo(t, w.Root, "seed.txt")
			task, err := store.CreateTask(w, "a-root", "p", "Crash-safe worker", store.TaskOpts{Accept: []string{"observable"}})
			if err != nil {
				t.Fatal(err)
			}
			if err := branchWithCommit(w.Root, taskBranch(task)); err != nil {
				t.Fatal(err)
			}
			first := newDriver(w, &fakeRunner{}, &Governor{})
			first.now = func() time.Time { return time.Unix(50, 0) }
			if !first.checkpointTaskPhase(task, phase) {
				t.Fatal(first.phaseErr)
			}

			journal, err := readPhaseJournal(w, "p")
			if err != nil {
				t.Fatal(err)
			}
			runner := &fakeRunner{}
			restarted := newDriver(w, runner, &Governor{})
			restarted.phases = journal
			restarted.cfg.pr = true
			restarted.cfg.landing.Mode = "pr"
			restarted.runCycle([]*store.Task{task})
			for _, call := range runner.calls {
				if len(call) > 0 && call[0] == "spawn" && slices.Contains(call, "--detach") {
					t.Fatalf("restart after %s spawned duplicate: %v", phase, runner.calls)
				}
			}
			checkpoint, ok := restarted.taskPhase(task)
			if !ok || !phaseAtLeast(checkpoint.Phase, phaseCommitted) {
				t.Fatalf("restart after %s did not continue through committed phase: %+v", phase, checkpoint)
			}
		})
	}
}

func TestWidthTwoPhaseAdvancementPreservesDistinctTaskRunProvenance(t *testing.T) {
	w := loopEnv(t)
	first, err := store.CreateTask(w, "a-root", "p", "First parallel task", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateTask(w, "a-root", "p", "Second parallel task", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range []procmon.Record{
		{RunID: "01WIDTH2TASKONE00000000001", Task: first.ID},
		{RunID: "01WIDTH2TASKTWO00000000002", Task: second.ID},
		// This is globally newest, reproducing the old LatestRunID corruption.
		{RunID: "01WIDTH2UNRELATED000000003", Task: "p/unrelated"},
	} {
		path := filepath.Join(w.RunDir(rec.RunID), "proc.txt")
		if err := procmon.WriteRecord(path, rec); err != nil {
			t.Fatal(err)
		}
	}
	d := newDriver(w, &fakeRunner{}, &Governor{})
	for _, task := range []*store.Task{first, second} {
		if !d.checkpointTaskPhase(task, phaseSpawned) || !d.checkpointTaskPhase(task, phaseWaited) || !d.checkpointTaskPhase(task, phaseCommitted) || !d.checkpointTaskPhase(task, phaseVerified) {
			t.Fatal(d.phaseErr)
		}
	}
	one, ok := d.taskPhase(first)
	if !ok {
		t.Fatal("first task checkpoint missing")
	}
	two, ok := d.taskPhase(second)
	if !ok {
		t.Fatal("second task checkpoint missing")
	}
	if one.RunID != "01WIDTH2TASKONE00000000001" || two.RunID != "01WIDTH2TASKTWO00000000002" || one.RunID == two.RunID {
		t.Fatalf("width-two task/run provenance collapsed: first=%+v second=%+v", one, two)
	}
}

func TestRestartAfterPushOrPRCreationSkipsCompletedRemoteSteps(t *testing.T) {
	for _, tc := range []struct {
		phase cyclePhase
		want  []string
	}{{phasePushed, []string{"pr"}}, {phasePRCreated, nil}} {
		t.Run(string(tc.phase), func(t *testing.T) {
			w := loopEnv(t)
			task, err := store.CreateTask(w, "a-root", "p", "Crash-safe PR", store.TaskOpts{})
			if err != nil {
				t.Fatal(err)
			}
			first := newDriver(w, &fakeRunner{}, &Governor{})
			if !first.checkpointTaskPhase(task, tc.phase) {
				t.Fatal(first.phaseErr)
			}
			journal, err := readPhaseJournal(w, "p")
			if err != nil {
				t.Fatal(err)
			}
			runner := &fakeRunner{}
			restarted := newDriver(w, runner, &Governor{})
			restarted.phases = journal
			if !restarted.queueTaskPR(task) {
				t.Fatal("idempotent PR recovery failed")
			}
			if !slices.Equal(runner.firstArgs(), tc.want) {
				t.Fatalf("calls after %s = %v, want %v", tc.phase, runner.firstArgs(), tc.want)
			}
		})
	}
}

func TestRestartAfterMergeAndAcceptanceDoesNotAcceptTwice(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Crash-safe acceptance", store.TaskOpts{Accept: []string{"observable"}})
	if err != nil {
		t.Fatal(err)
	}
	store.CheckAllAcceptance(task)
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, task, "done"); err != nil {
		t.Fatal(err)
	}
	first := newDriver(w, &fakeRunner{}, &Governor{})
	if !first.checkpointTaskPhase(task, phaseMerged) {
		t.Fatal(first.phaseErr)
	}
	journal, err := readPhaseJournal(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	restarted := newDriver(w, runner, &Governor{})
	restarted.phases = journal
	restarted.pendingAccept = []pendingAccept{{Seq: task.Seq, Branch: taskBranch(task), Generation: task.Generation(), GenerationSet: true}}
	stubOrchestrationGH(t, func(string, ...string) (string, error) { return `[{"state":"MERGED"}]`, nil })
	restarted.reconcilePendingAccepts()
	for _, call := range runner.calls {
		if len(call) > 0 && call[0] == "accept" {
			t.Fatalf("already-applied acceptance replayed: %v", runner.calls)
		}
	}
	checkpoint, ok := restarted.taskPhase(task)
	if !ok || checkpoint.Phase != phaseAccepted {
		t.Fatalf("accepted reality was not checkpointed: %+v", checkpoint)
	}
}

func TestRestartRestoresPRCIPendingAndMergedLandingWithoutNewWave(t *testing.T) {
	for _, phase := range []cyclePhase{phasePRCreated, phaseCIPending, phaseMerged} {
		t.Run(string(phase), func(t *testing.T) {
			w := loopEnv(t)
			task, err := store.CreateTask(w, "a-root", "p", "Restore external landing", store.TaskOpts{})
			if err != nil {
				t.Fatal(err)
			}
			first := newDriver(w, &fakeRunner{}, &Governor{})
			if !first.checkpointTaskPhase(task, phase) {
				t.Fatal(first.phaseErr)
			}
			journal, err := readPhaseJournal(w, "p")
			if err != nil {
				t.Fatal(err)
			}
			restarted := newDriver(w, &fakeRunner{}, &Governor{})
			restarted.phases = journal
			restarted.restoreLandingPhases()
			if len(restarted.pendingAccept) != 1 || restarted.pendingAccept[0].Seq != task.Seq {
				t.Fatalf("%s restart did not restore pending acceptance: %+v", phase, restarted.pendingAccept)
			}
			wantLand := phase != phaseMerged
			if (len(restarted.pendingLand) == 1) != wantLand {
				t.Fatalf("%s pending record hold = %v, want present=%t", phase, restarted.pendingLand, wantLand)
			}
		})
	}
}

func TestReadPhaseJournalRejectsTornOrUnknownState(t *testing.T) {
	w := loopEnv(t)
	path := phaseJournalFile(w, "p")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema":"loop-phase-journal/v999","version":999,"project":"p"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPhaseJournal(w, "p"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unknown phase schema did not fail closed: %v", err)
	}
}
