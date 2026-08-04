package eventlog

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// setup builds a throwaway workspace with one project and one open task.
func setup(t *testing.T) (*workspace.Workspace, *store.Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("create project: %v", err)
	}
	task, err := store.CreateTask(w, "a-root", "core", "do a thing", store.TaskOpts{})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	return w, task
}

// reopen simulates a mid-apply crash: the side effects committed but the event
// was never marked applied, so it is still pending on the next Sync.
func reopen(t *testing.T, path string) {
	t.Helper()
	d, err := mdstore.ReadFile(path)
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	d.Front.Set("applied", "false")
	if err := mdstore.WriteFile(path, d); err != nil {
		t.Fatalf("write event: %v", err)
	}
}

func TestSyncIsIdempotent(t *testing.T) {
	w, task := setup(t)
	always := func(string) bool { return true }

	claim, err := Append(w, "a-worker", model.EventClaim, task.Slug, "", "")
	if err != nil {
		t.Fatalf("append claim: %v", err)
	}
	finding, err := Append(w, "a-worker", model.EventFinding, task.Slug, "", "the bug\nlong body here")
	if err != nil {
		t.Fatalf("append finding: %v", err)
	}

	if _, err := Sync(w, "a-root", always); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// Re-open both events and sync again. The second pass must not duplicate
	// the claim log line or the finding note.
	reopen(t, claim.Path)
	reopen(t, finding.Path)
	if _, err := Sync(w, "a-root", always); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	notes, err := store.ListNotes(w, "core", model.NoteFinding)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 finding note after re-sync, got %d (duplicate on re-apply)", len(notes))
	}

	tk, err := store.FindTask(w, task.Slug)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	logSec, _ := tk.Doc.Section("Log")
	if n := strings.Count(logSec.Content, "claimed by a-worker"); n != 1 {
		t.Fatalf("expected 1 claim log line after re-sync, got %d", n)
	}
	if n := strings.Count(logSec.Content, "finding by a-worker"); n != 1 {
		t.Fatalf("expected 1 finding log line after re-sync, got %d", n)
	}
}

// setupWithAccept builds a workspace with one open task that carries the given
// acceptance criteria, so a propose:done can be verified against real boxes.
func setupWithAccept(t *testing.T, accept ...string) (*workspace.Workspace, *store.Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("create project: %v", err)
	}
	task, err := store.CreateTask(w, "a-root", "core", "do a thing", store.TaskOpts{Accept: accept})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	return w, task
}

// TestSyncProposeDoneRoutesThroughCloseTask is the task-284 regression guard: a
// non-owner's `task done` files an EventProposeStatus "propose: done", and when
// the owner Syncs it the task must close exactly like the owner's direct
// CloseTask — the "completed by" actuals stamp is written (not a bare MoveTask),
// so calibration's claim→completion span and doctor see the close. Before the
// fix, apply() moved the task to done via store.MoveTask directly, leaving no
// stamp: the task was excluded from calibration and flagged by doctor.
func TestSyncProposeDoneRoutesThroughCloseTask(t *testing.T) {
	w, task := setupWithAccept(t, "the thing works")
	always := func(string) bool { return true }

	// All acceptance boxes checked — the completing agent verified them.
	store.CheckAllAcceptance(task)
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("save acceptance: %v", err)
	}

	// A non-owner claims and then proposes done (the two events a non-owner's
	// claim + `task done` file). Oldest-first Sync applies claim then done.
	if _, err := Append(w, "a-worker", model.EventClaim, task.Slug, "", ""); err != nil {
		t.Fatalf("append claim: %v", err)
	}
	if _, err := Append(w, "a-worker", model.EventProposeStatus, task.Slug, "", "propose: done"); err != nil {
		t.Fatalf("append propose done: %v", err)
	}

	if _, err := Sync(w, "a-root", always); err != nil {
		t.Fatalf("sync: %v", err)
	}

	tk, err := store.FindTask(w, task.Slug)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	if tk.Status != model.StatusDone {
		t.Fatalf("expected task in done/ after propose:done sync, got %s", tk.Status)
	}
	// The canonical stamp calibration reads must be present — routed through
	// CloseTask, attributed to the proposing agent who did the work.
	if !store.LogHasStamp(tk, "completed by") {
		t.Fatalf("propose:done left no 'completed by' stamp — bypassed CloseTask")
	}
	logSec, _ := tk.Doc.Section("Log")
	if n := strings.Count(logSec.Content, "completed by a-worker"); n != 1 {
		t.Fatalf("expected exactly 1 'completed by a-worker' stamp, got %d", n)
	}
	if !store.LogHasStamp(tk, "claimed by") {
		t.Fatalf("expected a 'claimed by' stamp so the calibration span is measurable")
	}

	// Idempotency: a mid-apply crash re-runs the event. It must NOT append a
	// second "completed by" stamp (which would corrupt the span).
	pending, err := List(w, Query{About: task.Slug, Kinds: []model.EventKind{model.EventProposeStatus}})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 propose event, got %d", len(pending))
	}
	reopen(t, pending[0].Path)
	if _, err := Sync(w, "a-root", always); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	tk, err = store.FindTask(w, task.Slug)
	if err != nil {
		t.Fatalf("find task after re-sync: %v", err)
	}
	logSec, _ = tk.Doc.Section("Log")
	if n := strings.Count(logSec.Content, "completed by a-worker"); n != 1 {
		t.Fatalf("expected 1 'completed by' stamp after re-sync, got %d (not idempotent)", n)
	}
}

// TestSyncProposeDoneUnmetAcceptanceStaysPending guards the verification half of
// task 284: a propose:done on a task with an UNCHECKED acceptance box must NOT
// be moved to done — it mirrors the owner path's refusal (planning.go
// cmdTaskDone), leaving the event pending for a human. A 'done' no check
// supports must never land.
func TestSyncProposeDoneUnmetAcceptanceStaysPending(t *testing.T) {
	w, task := setupWithAccept(t, "the thing works")
	always := func(string) bool { return true }

	// Note: acceptance box is left UNCHECKED.
	propose, err := Append(w, "a-worker", model.EventProposeStatus, task.Slug, "", "propose: done")
	if err != nil {
		t.Fatalf("append propose done: %v", err)
	}

	res, err := Sync(w, "a-root", always)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if res.Applied != 0 {
		t.Fatalf("expected Sync to apply 0 events (unmet acceptance), applied %d", res.Applied)
	}

	tk, err := store.FindTask(w, task.Slug)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	if tk.Status == model.StatusDone {
		t.Fatalf("task with an unchecked acceptance box was moved to done/ — silent close")
	}
	if store.LogHasStamp(tk, "completed by") {
		t.Fatalf("unmet-acceptance task got a 'completed by' stamp — must not close")
	}

	// The event stays pending so a human resolves it — not silently dropped.
	pending, err := List(w, Query{About: task.Slug, Kinds: []model.EventKind{model.EventProposeStatus}, Pending: true})
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	found := false
	for _, e := range pending {
		if e.ID == propose.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("propose:done with unmet acceptance was consumed instead of left pending")
	}
}

// TestSyncLeavesAcceptProposalPending is the regression guard for the R5
// accept-propose/sync race: a read-only agent records its acceptance close as
// an EventComment whose body carries ProposePrefix. `dacli accept` is that
// comment's only consumer — it stays pending until the owner accepts. A Sync
// running in between (the documented owner path, and every supervise turn) must
// NOT consume it as a generic comment, or accept sees no pending proposal and
// the task never closes with no signal.
func TestSyncLeavesAcceptProposalPending(t *testing.T) {
	w, task := setup(t)
	always := func(string) bool { return true }

	body := ProposePrefix + " a-worker completed; proposing all acceptance boxes checked"
	proposal, err := Append(w, "a-worker", model.EventComment, task.Slug, "", body)
	if err != nil {
		t.Fatalf("append proposal: %v", err)
	}

	res, err := Sync(w, "a-root", always)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if res.Applied != 0 {
		t.Fatalf("expected Sync to apply 0 events, applied %d (proposal was consumed)", res.Applied)
	}

	// The proposal must still be pending so accept can find it.
	pending, err := List(w, Query{About: task.Slug, Kinds: []model.EventKind{model.EventComment}, Pending: true})
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	found := false
	for _, e := range pending {
		if e.ID == proposal.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("accept-propose comment was consumed by Sync — no longer pending for `dacli accept`")
	}

	// And it must not have leaked into the task Log as a generic comment.
	tk, err := store.FindTask(w, task.Slug)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	logSec, _ := tk.Doc.Section("Log")
	if strings.Contains(logSec.Content, ProposePrefix) {
		t.Fatalf("accept-propose body leaked into the task Log; Sync should have left it untouched")
	}
}

// TestSyncStillAppliesGenericComment guards the other side of the fork: a
// non-proposal EventComment must still be logged and marked applied, so the
// ProposePrefix skip does not silently swallow ordinary comments.
func TestSyncStillAppliesGenericComment(t *testing.T) {
	w, task := setup(t)
	always := func(string) bool { return true }

	comment, err := Append(w, "a-worker", model.EventComment, task.Slug, "", "just a note for the owner")
	if err != nil {
		t.Fatalf("append comment: %v", err)
	}

	res, err := Sync(w, "a-root", always)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if res.Applied != 1 {
		t.Fatalf("expected Sync to apply the generic comment, applied %d", res.Applied)
	}

	pending, err := List(w, Query{About: task.Slug, Pending: true})
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	for _, e := range pending {
		if e.ID == comment.ID {
			t.Fatalf("generic comment left pending — the ProposePrefix skip is over-matching")
		}
	}

	tk, err := store.FindTask(w, task.Slug)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	logSec, _ := tk.Doc.Section("Log")
	if !strings.Contains(logSec.Content, "just a note for the owner") {
		t.Fatalf("generic comment was not logged to the task")
	}
}
