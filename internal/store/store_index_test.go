package store

import (
	"fmt"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func indexWorkspace(t *testing.T) *workspace.Workspace {
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

func TestTaskIndexResolvesEveryRefForm(t *testing.T) {
	w := indexWorkspace(t)
	task, err := CreateTask(w, "a-root", "core", "index me", TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}

	idx, err := BuildTaskIndex(w)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	// Every ref form FindTask accepts must resolve to the same task through
	// the index.
	refs := []string{
		task.ID,
		task.Slug,
		fmt.Sprintf("%03d", task.Seq),
		fmt.Sprintf("%d", task.Seq),
		fmt.Sprintf("%03d-%s", task.Seq, task.Slug),
	}
	for _, ref := range refs {
		got, err := idx.Find(ref)
		if err != nil {
			t.Fatalf("index Find(%q): %v", ref, err)
		}
		if got.ID != task.ID {
			t.Fatalf("index Find(%q) = %s, want %s", ref, got.ID, task.ID)
		}
		// And it must agree with the one-shot FindTask.
		direct, err := FindTask(w, ref)
		if err != nil || direct.ID != task.ID {
			t.Fatalf("FindTask(%q) disagrees with index", ref)
		}
	}

	if _, err := idx.Find("no-such-task"); err == nil {
		t.Fatal("expected ErrNotFound for unknown ref")
	}
}

func TestTaskIndexReportsAmbiguity(t *testing.T) {
	w := indexWorkspace(t)
	if _, err := CreateProject(w, "a-root", "Other", "other", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	// Same slug in two projects: seq 1 in each → ref "1" and the slug are
	// ambiguous, exactly as FindTask treats it.
	if _, err := CreateTask(w, "a-root", "core", "dup", TaskOpts{}); err != nil {
		t.Fatalf("task: %v", err)
	}
	if _, err := CreateTask(w, "a-root", "other", "dup", TaskOpts{}); err != nil {
		t.Fatalf("task: %v", err)
	}

	idx, err := BuildTaskIndex(w)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if _, err := idx.Find("dup"); err == nil {
		t.Fatal("expected ambiguity error for a slug shared across projects")
	}
}
