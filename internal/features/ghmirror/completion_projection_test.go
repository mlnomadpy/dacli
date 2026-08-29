package ghmirror

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestIssueProjectionNamesImplementedUnlandedWithoutTerminalClosure(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "p", "Await landing", store.TaskOpts{Accept: []string{"verified"}})
	if err != nil {
		t.Fatal(err)
	}
	task.Doc.Front.Set("completion_state", "implemented-unlanded")
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	body := issueBody(w, task)
	if !strings.Contains(body, "State: `implemented-unlanded`") || statusLabel(task.Status) != "status:open" {
		t.Fatalf("GitHub projection conflated lifecycle: label=%s body=%q", statusLabel(task.Status), body)
	}
}
