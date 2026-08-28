package ghmirror

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestGitHubProjectionReportsTheSameDerivedAggregateProgress(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "aggregate-projection")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	parent, _ := store.CreateTask(w, "a-root", "core", "Milestone", store.TaskOpts{Accept: []string{"done"}})
	for _, title := range []string{"API", "UI"} {
		if _, err := store.CreateTask(w, "a-root", "core", title, store.TaskOpts{Parent: parent.ID, Accept: []string{title + " verified"}}); err != nil {
			t.Fatal(err)
		}
	}
	plan, _ := store.BuildAggregateRepairPlan(w, parent)
	if _, err := store.ApplyAggregateRepairPlan(w, parent, plan.ID); err != nil {
		t.Fatal(err)
	}
	parent, _ = store.FindTask(w, parent.ID)
	body := issueBody(w, parent)
	for _, want := range []string{"### Aggregate progress", "0/2 required children complete", "ready to close: false", plan.ChildIDs[0], plan.ChildIDs[1]} {
		if !strings.Contains(body, want) {
			t.Fatalf("projection missing %q:\n%s", want, body)
		}
	}
}
