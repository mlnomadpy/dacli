package store

import (
	"fmt"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// TaskSnapshot is a command/cycle-scoped task view. It is deliberately not a
// package cache: callers own its lifetime, invalidate it at the first mutation,
// and establish the next stable phase with Refresh. Find refuses an invalid
// snapshot so stale state cannot silently survive a mutation boundary.
type TaskSnapshot struct {
	w     *workspace.Workspace
	tasks []*Task
	index *TaskIndex
	valid bool
}

// LoadTaskSnapshot reads and indexes the task tree once for a stable phase.
func LoadTaskSnapshot(w *workspace.Workspace) (*TaskSnapshot, error) {
	s := &TaskSnapshot{w: w}
	if err := s.Refresh(); err != nil {
		return nil, err
	}
	return s, nil
}

// Tasks returns the loaded tasks while the snapshot is valid.
func (s *TaskSnapshot) Tasks() ([]*Task, error) {
	if !s.valid {
		return nil, fmt.Errorf("task snapshot is invalid; refresh at the next stable phase")
	}
	return s.tasks, nil
}

// Find resolves a reference from the phase's one index.
func (s *TaskSnapshot) Find(ref string) (*Task, error) {
	if !s.valid {
		return nil, fmt.Errorf("task snapshot is invalid; refresh at the next stable phase")
	}
	return s.index.Find(ref)
}

// Invalidate ends the stable phase before any mutation that can affect tasks.
func (s *TaskSnapshot) Invalidate() {
	s.valid = false
	s.tasks = nil
	s.index = nil
}

// Refresh starts the next stable phase from current durable state.
func (s *TaskSnapshot) Refresh() error {
	tasks, err := ListTasks(s.w, "", "")
	if err != nil {
		s.Invalidate()
		return err
	}
	s.tasks = tasks
	s.index = NewTaskIndex(tasks)
	s.valid = true
	return nil
}
