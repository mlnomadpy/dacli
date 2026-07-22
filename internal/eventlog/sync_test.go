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
