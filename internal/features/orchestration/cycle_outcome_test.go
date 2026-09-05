package orchestration

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestCycleOutcomeDistinguishesHealthyIdleAndFullyLanded(t *testing.T) {
	d := &driver{}
	idle := d.finishCycleOutcome(0, cycleRollup{})
	if idle.Schema != cycleOutcomeSchema || idle.Classification != "healthy-idle" || idle.err() != nil {
		t.Fatalf("idle=%+v err=%v", idle, idle.err())
	}
	landed := d.finishCycleOutcome(2, cycleRollup{Landed: 2})
	if landed.Classification != "healthy" || landed.Landed != 2 || landed.err() != nil {
		t.Fatalf("landed=%+v err=%v", landed, landed.err())
	}
}

func TestCycleOutcomeKeepsPolicyAndOperationalFailuresTyped(t *testing.T) {
	policy := &driver{}
	policy.recordCycleFailure("review", clikit.Refusedf("approval withheld"), "approval withheld")
	policyOutcome := policy.finishCycleOutcome(1, cycleRollup{ProducedNothing: 1})
	if policyOutcome.Classification != "degraded-zero-output" || clikit.ExitCode(policyOutcome.err()) != 3 || policyOutcome.Failures[0].Retryable {
		t.Fatalf("policy outcome=%+v err=%v", policyOutcome, policyOutcome.err())
	}

	operational := &driver{}
	operational.recordCycleFailure("ship", errors.New("remote unavailable"), "remote unavailable")
	operationalOutcome := operational.finishCycleOutcome(1, cycleRollup{Stalled: 1})
	if clikit.ExitCode(operationalOutcome.err()) != 1 || !operationalOutcome.Failures[0].Retryable {
		t.Fatalf("operational outcome=%+v err=%v", operationalOutcome, operationalOutcome.err())
	}
}

type zeroOutputRunner struct {
	fakeRunner
	w *workspace.Workspace
}

func (r *zeroOutputRunner) run(label string, args ...string) (string, error) {
	_, _ = r.fakeRunner.run(label, args...)
	if len(args) > 0 && args[0] == "spawn" {
		task, err := store.FindTask(r.w, argAfter(args, "--task"))
		if err != nil {
			return "", err
		}
		runID := "01ZEROOUTPUT00000000000000"
		dir := r.w.RunDir(runID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{RunID: runID, Task: task.ID, Outcome: "completed", Started: time.Now()}); err != nil {
			return "", err
		}
	}
	return "", nil
}

func TestLoopReturnsOperationalFailureAfterPersistingZeroOutputSpawn(t *testing.T) {
	w := loopEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "Produces no commit", store.TaskOpts{Accept: []string{"commit exists"}})
	if err != nil {
		t.Fatal(err)
	}
	d := newDriver(w, &zeroOutputRunner{w: w}, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	d.cfg.width = 1
	err = d.loop()
	if clikit.ExitCode(err) != 1 {
		t.Fatalf("zero-output loop err=%v code=%d", err, clikit.ExitCode(err))
	}
	st, readErr := readLoopState(w, "p")
	if readErr != nil || st.Outcome.Classification != "degraded-zero-output" || st.Outcome.ProducedNothing != 1 || len(st.Outcome.Failures) == 0 || st.Outcome.Failures[0].Phase != "commit" {
		t.Fatalf("persisted outcome=%+v readErr=%v", st.Outcome, readErr)
	}
	_ = task
}

func TestLoopStatusJSONExposesVersionedOutcomeAcrossRestart(t *testing.T) {
	w := loopEnv(t)
	d := newDriver(w, &fakeRunner{}, &Governor{})
	d.lastRollup = cycleRollup{Landed: 1}
	d.lastOutcome = d.finishCycleOutcome(1, d.lastRollup)
	if err := d.saveState("proceed", "cycle complete", 0); err != nil {
		t.Fatal(err)
	}
	restarted, err := readLoopState(w, "p")
	if err != nil || restarted.Outcome.Schema != cycleOutcomeSchema || restarted.Outcome.Landed != 1 {
		t.Fatalf("restart outcome=%+v err=%v", restarted.Outcome, err)
	}
	var out bytes.Buffer
	ctx := &clikit.Ctx{Cwd: w.Root, Stdout: &out, Stderr: &bytes.Buffer{}, JSON: true}
	if err := cmdLoopStatus(ctx, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Outcome cycleOutcome `json:"cycle_outcome"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil || payload.Outcome.Schema != cycleOutcomeSchema || payload.Outcome.Classification != "healthy" {
		t.Fatalf("status=%s decode=%v outcome=%+v", out.String(), err, payload.Outcome)
	}
}
