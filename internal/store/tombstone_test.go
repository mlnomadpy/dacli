package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func tombstoneWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

// Seq allocation promises "monotonic-never-reuse". The git ceiling delivers it
// for any seq ever COMMITTED, but a workspace that records to its own branch
// has .dacli gitignored, so a task created AND removed between two ships was
// never committed — the seq came back, and a live agent's ref resolved to a
// DIFFERENT task with different content (issue #433, reported by the agent it
// happened to).
func TestARemovedSeqIsNeverHandedOutAgain(t *testing.T) {
	w := tombstoneWS(t)

	first, err := CreateTask(w, "a-root", "p", "First", TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Seq != 1 {
		t.Fatalf("fixture seq = %d, want 1", first.Seq)
	}
	if err := RemoveTask(w, first); err != nil {
		t.Fatal(err)
	}

	second, err := CreateTask(w, "a-root", "p", "Second", TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if second.Seq == first.Seq {
		t.Fatalf("seq %d was handed out twice; a stale ref now resolves to a different task", second.Seq)
	}

	// And the stale ref resolves to NOTHING rather than to the new task —
	// which is the failure mode: an agent asked for its own task and got
	// someone else's.
	if got, err := FindTask(w, "001"); err == nil {
		t.Errorf("the removed ref 001 still resolves, to %q", got.Title)
	}
}

// The tombstone is a record, not a bare marker. Removal is the one operation
// that destroys history, so what it destroyed is written down — a reader who
// finds a gap in the numbering gets an answer instead of a mystery.
func TestTheTombstoneRecordsWhatWasRemoved(t *testing.T) {
	w := tombstoneWS(t)
	task, err := CreateTask(w, "a-root", "p", "A task that should never have existed", TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	id := task.ID
	if err := RemoveTask(w, task); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(w.TombstonesDir("p"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected exactly one tombstone, got %v (%v)", entries, err)
	}
	raw, err := os.ReadFile(filepath.Join(w.TombstonesDir("p"), entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{id, "A task that should never have existed", "001"} {
		if !strings.Contains(body, want) {
			t.Errorf("the tombstone must record %q:\n%s", want, body)
		}
	}
}

// A tombstone is NOT a task. listTasksRaw iterates model.AllStatuses
// explicitly, so the removed/ sibling must never be read back as backlog — a
// removed task reappearing in `task list` would be worse than the reuse this
// fixes.
func TestTombstonesAreNotListedAsTasks(t *testing.T) {
	w := tombstoneWS(t)
	task, err := CreateTask(w, "a-root", "p", "Removed", TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := RemoveTask(w, task); err != nil {
		t.Fatal(err)
	}
	got, err := ListTasks(w, "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("a tombstone was listed as a task: %v", got)
	}
}
