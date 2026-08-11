// The broken-task-file record is process-global and additive by construction —
// listTasksRaw is called from everywhere and threading a second return value
// through every caller would be a far larger change than the bug it closes.
// That construction has one consequence worth pinning down: the record must
// still describe the CURRENT workspace. A file that was broken and has since
// been repaired, or deleted, is no longer a finding, and doctor reporting it
// would send a reader after a problem that is already gone.
package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// brokenFixture creates one healthy task, then overwrites its file with the
// realistic corruption: a git conflict marker inside the frontmatter, which is
// what an agent-written .dacli produces on a bad merge.
func brokenFixture(t *testing.T) (*workspace.Workspace, string, []byte) {
	t.Helper()
	w := runtimeWorkspace(t)
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTask(w, "a-root", "core", "A task that will be corrupted", TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	good, err := os.ReadFile(task.Path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	corrupt := "---\n<<<<<<< HEAD\ntitle: A task that will be corrupted\n=======\ntitle: something else\n>>>>>>> other\n"
	if err := os.WriteFile(task.Path, []byte(corrupt), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return w, task.Path, good
}

func brokenRecorded(t *testing.T, path string) bool {
	t.Helper()
	for _, b := range BrokenTaskFiles() {
		if b.Path == path {
			return true
		}
	}
	return false
}

// TestBrokenTaskFileIsRecorded is the premise the other two tests rest on: an
// unparseable file drops out of the listing (status is folder position, so it
// cannot be half-read) and is recorded instead of vanishing.
func TestBrokenTaskFileIsRecorded(t *testing.T) {
	w, path, _ := brokenFixture(t)
	tasks, err := ListTasks(w, "", "")
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("a file that does not parse must not appear in the listing, got %d task(s)", len(tasks))
	}
	if !brokenRecorded(t, path) {
		t.Fatalf("expected %s in the broken record, got %v", path, BrokenTaskFiles())
	}
}

// TestRepairedTaskFileLeavesTheBrokenRecord: parsing successfully is the only
// signal that the corruption is gone, so the listing loop must clear the entry
// there. Without it, doctor in any process that outlives one command names a
// problem the owner already fixed.
func TestRepairedTaskFileLeavesTheBrokenRecord(t *testing.T) {
	w, path, good := brokenFixture(t)
	if _, err := ListTasks(w, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if !brokenRecorded(t, path) {
		t.Fatalf("setup: expected %s recorded broken first", path)
	}

	if err := os.WriteFile(path, good, 0o644); err != nil {
		t.Fatalf("repair: %v", err)
	}
	tasks, err := ListTasks(w, "", "")
	if err != nil {
		t.Fatalf("ListTasks after repair: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("repaired task should list again, got %d", len(tasks))
	}
	if brokenRecorded(t, path) {
		t.Fatalf("a repaired file must leave the broken record, still present: %v", BrokenTaskFiles())
	}
}

// TestDeletedTaskFileLeavesTheBrokenRecord covers the path a successful parse
// can never reach: a broken file that is removed is never parsed again, so the
// listing loop has no chance to clear it. The reader has to drop it.
func TestDeletedTaskFileLeavesTheBrokenRecord(t *testing.T) {
	w, path, _ := brokenFixture(t)
	if _, err := ListTasks(w, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if !brokenRecorded(t, path) {
		t.Fatalf("setup: expected %s recorded broken first", path)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if brokenRecorded(t, path) {
		t.Fatalf("a deleted file must leave the broken record, still present: %v", BrokenTaskFiles())
	}
}

// TestBrokenTaskFileSeqStaysVisibleToAllocation is why the record exists at
// all: an unparseable file is missing from every listing, so nothing stops the
// allocator reissuing its NNN and producing two different tasks under one ref.
func TestBrokenTaskFileSeqStaysVisibleToAllocation(t *testing.T) {
	w, path, _ := brokenFixture(t)
	if _, err := ListTasks(w, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	next, err := CreateTask(w, "a-root", "core", "Work filed after the corruption", TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if got := filepath.Base(path); got == filepath.Base(next.Path) {
		t.Fatalf("allocator reissued the corrupted file's name %s", got)
	}
	if next.Seq <= 1 {
		t.Fatalf("seq %d reuses the corrupted task's number; allocation must stay above it", next.Seq)
	}
	if next.Status != model.StatusOpen {
		t.Fatalf("new task landed in %s", next.Status)
	}
}
