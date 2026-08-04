package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func dedupWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	return w
}

// planting a stale duplicate: copy a task file into a second status folder,
// leaving the original in place — the exact 026 drift.
func plantStaleCopy(t *testing.T, w *workspace.Workspace, task *Task, into model.Status) string {
	t.Helper()
	data, err := os.ReadFile(task.Path)
	if err != nil {
		t.Fatalf("read src: %v", err)
	}
	dir := w.TasksDir(task.Project, into)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dst := filepath.Join(dir, filepath.Base(task.Path))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write dup: %v", err)
	}
	return dst
}

// FindTask must resolve cleanly when a stale duplicate exists, preferring the
// terminal (done) copy — not error "ambiguous" on the same task twice.
func TestFindTaskDedupsStaleDuplicate(t *testing.T) {
	w := dedupWorkspace(t)
	task, err := CreateTask(w, "a-root", "core", "dup me", TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	// The authoritative copy lives in done/; the stale one stays in open/.
	if err := MoveTask(w, task, model.StatusDone); err != nil {
		t.Fatalf("move: %v", err)
	}
	plantStaleCopy(t, w, task, model.StatusOpen)

	got, err := FindTask(w, "1")
	if err != nil {
		t.Fatalf("FindTask on stale duplicate errored (want clean resolve): %v", err)
	}
	if got.Status != model.StatusDone {
		t.Fatalf("FindTask resolved to %s copy, want the authoritative done copy", got.Status)
	}

	// ListTasks must yield the task exactly once.
	all, err := ListTasks(w, "core", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	n := 0
	for _, x := range all {
		if x.Seq == task.Seq {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("ListTasks yielded seq %d %d times, want 1", task.Seq, n)
	}
}

// MoveTask must guarantee the source-status copy is gone AND that a pre-existing
// stale copy in a third folder cannot survive the move.
func TestMoveTaskLeavesNoStaleCopy(t *testing.T) {
	w := dedupWorkspace(t)
	task, err := CreateTask(w, "a-root", "core", "move me", TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	// Plant a stale copy in blocked/ before moving open/ -> done/.
	stale := plantStaleCopy(t, w, task, model.StatusBlocked)

	if err := MoveTask(w, task, model.StatusDone); err != nil {
		t.Fatalf("move: %v", err)
	}

	// Source (open/) removed by the rename.
	if _, err := os.Stat(filepath.Join(w.TasksDir("core", model.StatusOpen), filepath.Base(task.Path))); !os.IsNotExist(err) {
		t.Fatalf("source open copy survived the move")
	}
	// The pre-existing stale copy in blocked/ is swept.
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale blocked copy survived the move")
	}
	// Exactly one copy remains — in done/.
	if dups, _ := DuplicateTaskFiles(w); len(dups) != 0 {
		t.Fatalf("move left a duplicate: %+v", dups)
	}
}

// DuplicateTaskFiles must name a task present in two status folders.
func TestDuplicateTaskFilesReportsDrift(t *testing.T) {
	w := dedupWorkspace(t)
	task, err := CreateTask(w, "a-root", "core", "dup drift", TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if dups, _ := DuplicateTaskFiles(w); len(dups) != 0 {
		t.Fatalf("clean workspace reported duplicates: %+v", dups)
	}

	plantStaleCopy(t, w, task, model.StatusDone)

	dups, err := DuplicateTaskFiles(w)
	if err != nil {
		t.Fatalf("duplicates: %v", err)
	}
	if len(dups) != 1 {
		t.Fatalf("want 1 duplicate group, got %d", len(dups))
	}
	if dups[0].Seq != task.Seq || len(dups[0].Paths) != 2 {
		t.Fatalf("duplicate group wrong: %+v", dups[0])
	}
}
