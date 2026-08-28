package planning

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestTaskAggregatePreviewApplyAndListJSONShareDerivedState(t *testing.T) {
	w, _ := taskAddEnv(t)
	parent, _ := store.CreateTask(w, "a-root", "p", "Milestone", store.TaskOpts{Accept: []string{"done"}})
	for _, title := range []string{"API", "UI", "tests", "docs"} {
		if _, err := store.CreateTask(w, "a-root", "p", title, store.TaskOpts{Parent: parent.ID, Accept: []string{title}}); err != nil {
			t.Fatal(err)
		}
	}
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Cwd: w.Root, Stdout: out, Stderr: &bytes.Buffer{}, JSON: true}
	if err := cmdTaskAggregate(ctx, []string{parent.ID, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	var plan store.AggregateRepairPlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil || plan.ID == "" || len(plan.ChildIDs) != 4 {
		t.Fatalf("preview = %#v err=%v output=%s", plan, err, out)
	}
	tasks, _ := store.ListTasks(w, "p", "")
	if len(tasks) != 5 || tasks[0].IsAggregate() {
		t.Fatal("preview mutated the task graph")
	}
	out.Reset()
	if err := cmdTaskAggregate(ctx, []string{parent.ID, "--apply", plan.ID}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := cmdTaskListJSON(ctx, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}
	var rows []taskJSON
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if rows[0].Kind != store.TaskKindAggregate || rows[0].Aggregate == nil || rows[0].Aggregate.Required != 4 || rows[0].Aggregate.ReadyToClose {
		t.Fatalf("aggregate JSON row = %#v", rows[0])
	}
}

func TestTaskDecomposeRequiresExplicitApplyAndRejectsChangedPlan(t *testing.T) {
	w, _ := taskAddEnv(t)
	parent, _ := store.CreateTask(w, "a-root", "p", "Oversized", store.TaskOpts{Estimate: "8,13,21", Accept: []string{"internal/a/a.go complete", "internal/b/b.go complete"}})
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Cwd: w.Root, Stdout: out, Stderr: &bytes.Buffer{}, JSON: true}
	if err := cmdTaskDecompose(ctx, []string{parent.ID, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	var plan store.DecompositionPlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	parent.Doc.SetSection("Acceptance", "- [ ] internal/a/a.go changed\n- [ ] internal/b/b.go complete\n")
	if err := store.SaveTask(parent); err != nil {
		t.Fatal(err)
	}
	err := cmdTaskDecompose(ctx, []string{parent.ID, "--apply", plan.ID})
	if err == nil || !strings.Contains(err.Error(), "plan changed") {
		t.Fatalf("changed plan apply = %v", err)
	}
	tasks, _ := store.ListTasks(w, "p", "")
	if len(tasks) != 1 {
		t.Fatalf("stale apply partially created %d tasks", len(tasks)-1)
	}
}
