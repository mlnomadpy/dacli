package store

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func lockWS(t *testing.T) (*workspace.Workspace, *Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "lk")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	task, err := CreateTask(w, "a-root", "core", "the work", TaskOpts{Accept: []string{"it works"}})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	return w, task
}

// appendLog is the read-modify-write shape every task mutation has: take the
// doc read earlier, add a line, rewrite the WHOLE file.
func appendLog(t *Task, line string) {
	sec, _ := t.Doc.Section("Log")
	t.Doc.SetSection("Log", sec.Content+"- "+line+"\n")
}

// The lost update, reproduced. Two holders of the same task each append a
// line; without serialization the second rewrite drops the first's line
// entirely — and in the real system the event that produced it is already
// marked applied, so nothing ever restores it.
func TestWithTaskSerializesConcurrentAppends(t *testing.T) {
	w, task := lockWS(t)

	// Two independent readers, exactly as two dacli processes would be: each
	// loaded the task before either wrote.
	a, err := FindTask(w, task.Slug)
	if err != nil {
		t.Fatal(err)
	}
	b, err := FindTask(w, task.Slug)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for _, tc := range []struct {
		holder *Task
		line   string
	}{{a, "claimed by a-one"}, {b, "claimed by a-two"}} {
		wg.Add(1)
		go func(h *Task, line string) {
			defer wg.Done()
			if err := WithTask(w, h, func(fresh *Task) error {
				appendLog(fresh, line)
				return SaveTask(fresh)
			}); err != nil {
				t.Errorf("WithTask: %v", err)
			}
		}(tc.holder, tc.line)
	}
	wg.Wait()

	got, err := FindTask(w, task.Slug)
	if err != nil {
		t.Fatal(err)
	}
	sec, _ := got.Doc.Section("Log")
	for _, want := range []string{"claimed by a-one", "claimed by a-two"} {
		if !strings.Contains(sec.Content, want) {
			t.Errorf("log lost %q — a durable record was silently erased:\n%s", want, sec.Content)
		}
	}
}

// The re-read is the load-bearing half: locking alone would still let a caller
// write back the stale doc it captured before waiting. fn must receive the
// state as it is NOW.
func TestWithTaskHandsFnTheCurrentStateNotTheCapturedOne(t *testing.T) {
	w, task := lockWS(t)
	stale, err := FindTask(w, task.Slug)
	if err != nil {
		t.Fatal(err)
	}

	// Another process writes while our caller holds a pre-write copy.
	other, _ := FindTask(w, task.Slug)
	appendLog(other, "written by the other process")
	if err := SaveTask(other); err != nil {
		t.Fatal(err)
	}

	if err := WithTask(w, stale, func(fresh *Task) error {
		sec, _ := fresh.Doc.Section("Log")
		if !strings.Contains(sec.Content, "written by the other process") {
			t.Errorf("fn got a stale doc; the other process's line is missing:\n%s", sec.Content)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// A status change renames the file between folders, so the caller's path goes
// stale — including within one caller's own loop (a claim moves the task, then
// the next event reads it back). The lock must find the task where it IS.
func TestWithTaskFollowsATaskThatMovedStatus(t *testing.T) {
	w, task := lockWS(t)
	held, err := FindTask(w, task.Slug)
	if err != nil {
		t.Fatal(err)
	}
	oldPath := held.Path

	mover, _ := FindTask(w, task.Slug)
	if err := MoveTask(w, mover, model.StatusActive); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("precondition: the task should have moved off %s", oldPath)
	}

	if err := WithTask(w, held, func(fresh *Task) error {
		if fresh.Status != model.StatusActive {
			t.Errorf("fn got status %s, want active — the lock read a stale location", fresh.Status)
		}
		appendLog(fresh, "still reachable after the move")
		return SaveTask(fresh)
	}); err != nil {
		t.Fatal(err)
	}

	got, err := FindTask(w, task.Slug)
	if err != nil {
		t.Fatal(err)
	}
	sec, _ := got.Doc.Section("Log")
	if !strings.Contains(sec.Content, "still reachable after the move") {
		t.Errorf("the write did not land on the moved task:\n%s", sec.Content)
	}
	// And the caller's own copy is current, so its next mutation is not stale.
	if held.Status != model.StatusActive {
		t.Errorf("caller's task still reads %s; a later mutation would build on stale state", held.Status)
	}
}

// The lock file must not survive a completed mutation, or the next caller
// waits out the timeout for nothing.
func TestWithTaskReleasesItsLock(t *testing.T) {
	w, task := lockWS(t)
	held, _ := FindTask(w, task.Slug)
	if err := WithTask(w, held, func(*Task) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(taskLockPath(w, held)); !os.IsNotExist(err) {
		t.Errorf("lock file survived a completed mutation (stat err = %v)", err)
	}
	// And a second acquisition succeeds immediately.
	done := make(chan error, 1)
	go func() { done <- WithTask(w, held, func(*Task) error { return nil }) }()
	if err := <-done; err != nil {
		t.Errorf("second WithTask failed: %v", err)
	}
}

var _ = mdstore.Doc{}
