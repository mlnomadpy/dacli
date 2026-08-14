package store

import (
	"errors"
	"fmt"
	"strings"
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
	_, ambiguous := idx.Find("dup")
	if ambiguous == nil {
		t.Fatal("expected ambiguity error for a slug shared across projects")
	}

	// Ambiguity guidance is an API surface: every suggested ref must be valid
	// input to the resolver that emitted it (issue #628).
	parts := strings.SplitN(ambiguous.Error(), ": ", 2)
	if len(parts) != 2 {
		t.Fatalf("ambiguity error has no suggestions: %v", ambiguous)
	}
	for _, suggestion := range strings.Split(parts[1], ", ") {
		got, err := idx.Find(suggestion)
		if err != nil {
			t.Fatalf("suggested ref %q does not round-trip: %v", suggestion, err)
		}
		if got.Project+"/"+fmt.Sprintf("%03d-%s", got.Seq, got.Slug) != suggestion {
			t.Fatalf("suggested ref %q resolved to %s/%03d-%s", suggestion, got.Project, got.Seq, got.Slug)
		}
	}
}

func TestTaskResolversAcceptProjectQualifiedRefs(t *testing.T) {
	w := indexWorkspace(t)
	if _, err := CreateProject(w, "a-root", "Other", "other", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	core, err := CreateTask(w, "a-root", "core", "dup", TaskOpts{})
	if err != nil {
		t.Fatalf("core task: %v", err)
	}
	other, err := CreateTask(w, "a-root", "other", "dup", TaskOpts{})
	if err != nil {
		t.Fatalf("other task: %v", err)
	}
	idx, err := BuildTaskIndex(w)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	refs := []string{
		"core/1",
		"core/001",
		"core/001-dup",
		"core/dup",
	}
	for _, ref := range refs {
		for name, find := range map[string]func(string) (*Task, error){
			"FindTask": func(ref string) (*Task, error) { return FindTask(w, ref) },
			"index":    idx.Find,
		} {
			got, err := find(ref)
			if err != nil {
				t.Fatalf("%s(%q): %v", name, ref, err)
			}
			if got.ID != core.ID {
				t.Fatalf("%s(%q) = %s/%s, want core/%s", name, ref, got.Project, got.Slug, core.Slug)
			}
		}
	}
	if got, err := idx.Find("other/dup"); err != nil || got.ID != other.ID {
		t.Fatalf("qualified lookup crossed project boundary: got %#v, err %v", got, err)
	}
	if got, err := FindTask(w, core.ID); err != nil || got.ID != core.ID {
		t.Fatalf("bare globally unique ID is not backward compatible: got %#v, err %v", got, err)
	}
}

func TestTaskResolversDistinguishUnknownProjectAndTask(t *testing.T) {
	w := indexWorkspace(t)
	if _, err := CreateTask(w, "a-root", "core", "exists", TaskOpts{}); err != nil {
		t.Fatalf("task: %v", err)
	}
	idx, err := BuildTaskIndex(w)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	for name, find := range map[string]func(string) (*Task, error){
		"FindTask": func(ref string) (*Task, error) { return FindTask(w, ref) },
		"index":    idx.Find,
	} {
		_, projectErr := find("missing/exists")
		_, taskErr := find("core/missing")
		var projectNF, taskNF ErrNotFound
		if !errors.As(projectErr, &projectNF) || projectNF.Ref != "project/missing" {
			t.Errorf("%s unknown project error = %v", name, projectErr)
		}
		if !errors.As(taskErr, &taskNF) || taskNF.Ref != "project/core/task/missing" {
			t.Errorf("%s unknown task error = %v", name, taskErr)
		}
	}
}
