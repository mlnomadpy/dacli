package planning

import (
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
