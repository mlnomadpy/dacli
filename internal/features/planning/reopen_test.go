package planning

import (
	"github.com/mlnomadpy/dacli/internal/procmon"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func closedTask(t *testing.T, w *workspace.Workspace, title string) *store.Task {
	t.Helper()
	tk, err := store.CreateTask(w, "a-root", "p", title, store.TaskOpts{Accept: []string{"one", "two"}})
	if err != nil {
		t.Fatal(err)
	}
	store.CheckAllAcceptance(tk)
	if err := store.CloseTask(w, tk, "a-root"); err != nil {
		t.Fatal(err)
	}
	found, err := store.FindTask(w, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// Closing was a one-way door: a task force-accepted by mistake could only be
// corrected by editing the markdown store by hand — which is exactly what
// happened to tasks 336 and 339 when `accept --force` ran over a batch nobody
// read. The tool's product is a record and it had no command to fix the record
// (dacli 340).
func TestTaskReopenClearsTheBoxesAndRecordsWhy(t *testing.T) {
	w, ctx := taskAddEnv(t)
	tk := closedTask(t, w, "A task closed by mistake")
	if tk.Status != model.StatusDone {
		t.Fatalf("fixture is %s, not done — this test would measure nothing", tk.Status)
	}

	if err := cmdTaskReopen(ctx, []string{tk.ID, "--reason", "the work was never done"}); err != nil {
		t.Fatalf("reopen: %v", err)
	}

	got, err := store.FindTask(w, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusOpen {
		t.Errorf("status = %s, want open", got.Status)
	}
	// The boxes are a claim that the work was verified; a reopen says that
	// claim was wrong, so leaving them checked keeps the false record.
	sec, _ := got.Doc.Section("Acceptance")
	if strings.Contains(strings.ToLower(sec.Content), "- [x]") {
		t.Errorf("a checked acceptance box survived the reopen:\n%s", sec.Content)
	}
	// And the reason is in the log, or the next reader has a mystery.
	logSec, _ := got.Doc.Section("Log")
	if !strings.Contains(logSec.Content, "the work was never done") {
		t.Errorf("the reopen reason is not in the log:\n%s", logSec.Content)
	}
	if !strings.Contains(logSec.Content, "reopened by") {
		t.Errorf("the log does not record the reopen:\n%s", logSec.Content)
	}
}

// A reopen with no reason is a mystery later, and an already-open task has
// nothing to reopen — both refuse rather than doing something plausible.
func TestTaskReopenRefusesWithoutAReasonOrOnAnOpenTask(t *testing.T) {
	w, ctx := taskAddEnv(t)
	tk := closedTask(t, w, "Another closed task")

	err := cmdTaskReopen(ctx, []string{tk.ID})
	if err == nil || clikit.ExitCode(err) != 2 {
		t.Errorf("a reopen with no --reason must be a usage error, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "reason") {
		t.Errorf("the refusal must name what is missing: %v", err)
	}

	if err := cmdTaskReopen(ctx, []string{tk.ID, "--reason", "ok"}); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	// Second time: already open.
	again := cmdTaskReopen(ctx, []string{tk.ID, "--reason", "ok"})
	if again == nil {
		t.Error("reopening an already-open task must refuse, not silently succeed")
	}
}

// Removal is for a task whose EXISTENCE was the mistake — a probe, a
// duplicate. It must not become a way to erase work that happened, and must
// not leave a dangling reference behind.
func TestTaskRmRefusesDoneWorkAndReferencedTasks(t *testing.T) {
	w, ctx := taskAddEnv(t)

	probe, err := store.CreateTask(w, "a-root", "p", "A probe that should not exist", store.TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdTaskRm(ctx, []string{probe.ID}); err != nil {
		t.Fatalf("removing an open, unreferenced task should work: %v", err)
	}
	if _, err := store.FindTask(w, probe.ID); err == nil {
		t.Error("the task survived removal")
	}

	// A done task carries a record of work that happened.
	done := closedTask(t, w, "Real work that landed")
	err = cmdTaskRm(ctx, []string{done.ID})
	if err == nil || clikit.ExitCode(err) != 3 {
		t.Errorf("removing a DONE task must be refused (exit 3), got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "reopen") {
		t.Errorf("the refusal must point at the right tool: %v", err)
	}

	// A dangling dependency fails far from the deletion that caused it.
	base, err := store.CreateTask(w, "a-root", "p", "A base task", store.TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(w, "a-root", "p", "A dependent task", store.TaskOpts{
		Accept: []string{"x"}, DependsOn: []string{base.ID},
	}); err != nil {
		t.Fatal(err)
	}
	err = cmdTaskRm(ctx, []string{base.ID, "--force"})
	if err == nil {
		t.Error("removing a task something depends on must refuse, even with --force")
	} else if !strings.Contains(err.Error(), "referenced") {
		t.Errorf("the refusal must say what still points at it: %v", err)
	}
}

// A LIVE agent holding a task outranks every other removal check. Removing it
// leaves that agent working a ref that resolves to nothing — or worse, to a
// DIFFERENT task, because a freed seq is handed out again.
//
// Reported from inside the failure: an estimator mid-investigation watched
// `dacli task show 344` start returning someone else's finished task, with no
// signal that its own had been deleted (issue #433). The removal checks
// searched the RECORD — events and notes — and never .dacli/runs, so a task
// referenced by a running process read as unreferenced.
func TestTaskRmRefusesWhileALiveAgentHoldsTheTask(t *testing.T) {
	w, ctx := taskAddEnv(t)
	task, err := store.CreateTask(w, "a-root", "p", "A task an agent is working", store.TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}

	// A run record for a process that IS alive: this test's own pid, which is
	// the only pid it can be sure about.
	runID := "01LIVERUN0000000000000000"
	dir := filepath.Join(w.RunsDir(), runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	self := os.Getpid()
	start, _ := procmon.ProcStart(self)
	rec := procmon.Record{
		RunID: runID, Child: "a-estimator-live", Task: task.ID,
		PID: self, PGID: self, PIDStart: start,
	}
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}

	err = cmdTaskRm(ctx, []string{task.ID, "--force"})
	if err == nil {
		t.Fatal("a task held by a LIVE agent was removed — that agent's ref now resolves to nothing")
	}
	// --force must not get past this one: the alternative is a run that cannot
	// be made correct.
	if !strings.Contains(err.Error(), "a-estimator-live") {
		t.Errorf("the refusal must name the agent holding it: %v", err)
	}
	if _, ferr := store.FindTask(w, task.ID); ferr != nil {
		t.Errorf("the task was deleted despite the refusal: %v", ferr)
	}

	// With the run gone, removal works again — the guard is about liveness,
	// not about ever having been claimed.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := cmdTaskRm(ctx, []string{task.ID}); err != nil {
		t.Errorf("removal should succeed once no live agent holds it: %v", err)
	}
}
