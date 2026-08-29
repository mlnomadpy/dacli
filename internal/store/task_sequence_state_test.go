package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func sequenceStateWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "sequence-state")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestTaskSequenceStateRebuildsAfterManualAndMalformedDrift(t *testing.T) {
	w := sequenceStateWorkspace(t)
	first, err := CreateTask(w, "a-root", "core", "first", TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Seq != 1 {
		t.Fatalf("first seq = %d", first.Seq)
	}

	// A manually introduced filename changes the canonical directory
	// observation. It need not parse: allocation identity lives in the name.
	manual := filepath.Join(w.TasksDir("core", model.StatusOpen), "125-manual.md")
	if err := os.WriteFile(manual, []byte("not frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := CreateTask(w, "a-root", "core", "after manual drift", TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Seq != 126 {
		t.Fatalf("manual filename ceiling was ignored: seq=%d, want 126", second.Seq)
	}

	if err := os.WriteFile(taskSequenceStatePath(w, "core"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := CreateTask(w, "a-root", "core", "after malformed state", TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if third.Seq != 127 {
		t.Fatalf("malformed acceleration state did not rebuild: seq=%d, want 127", third.Seq)
	}
}

func TestTaskSequenceStateHotPathAdvancesDurably(t *testing.T) {
	w := sequenceStateWorkspace(t)
	for want := 1; want <= 20; want++ {
		task, err := CreateTask(w, "a-root", "core", "task", TaskOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if task.Seq != want {
			t.Fatalf("task %d seq = %d", want, task.Seq)
		}
	}
	raw, err := os.ReadFile(taskSequenceStatePath(w, "core"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" {
		t.Fatal("durable task sequence state is empty")
	}
}
