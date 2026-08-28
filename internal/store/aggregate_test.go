package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func aggregateFixture(t *testing.T) (*workspace.Workspace, *Task, []*Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "aggregate")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	parent, err := CreateTask(w, "a-root", "core", "Build the product", TaskOpts{Accept: []string{"all child work is complete"}, Estimate: "8,13,21"})
	if err != nil {
		t.Fatal(err)
	}
	children := []*Task{}
	for _, title := range []string{"API", "UI", "tests", "docs"} {
		child, createErr := CreateTask(w, "a-root", "core", title, TaskOpts{Parent: parent.ID, Accept: []string{title + " verified"}, Estimate: "1,2,3"})
		if createErr != nil {
			t.Fatal(createErr)
		}
		children = append(children, child)
	}
	return w, parent, children
}

func TestAggregateParentNeverAppearsInReadyFrontierAndOrdinaryParentStillDoes(t *testing.T) {
	w, parent, children := aggregateFixture(t)
	plan, err := BuildAggregateRepairPlan(w, parent)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyAggregateRepairPlan(w, parent, plan.ID); err != nil {
		t.Fatal(err)
	}
	// Readiness must enforce task kind directly, rather than relying on the
	// repair's dependency edges as an accidental second-order exclusion. A
	// hand-migrated aggregate may carry only its stable child list.
	parent, _ = FindTask(w, parent.ID)
	parent.Doc.Front.SetList("depends_on", nil)
	if err := SaveTask(parent); err != nil {
		t.Fatal(err)
	}
	tasks, _ := ListTasks(w, "core", "")
	frontier := ReadyFrontier(tasks)
	if len(frontier.Ready) != 4 {
		t.Fatalf("ready = %d, want four eligible children: %#v", len(frontier.Ready), frontier.Ready)
	}
	for _, ready := range frontier.Ready {
		if ready.ID == parent.ID {
			t.Fatal("aggregate parent was returned as directly assignable")
		}
	}

	w2, ordinary, ordinaryChildren := aggregateFixture(t)
	tasks, _ = ListTasks(w2, "core", "")
	frontier = ReadyFrontier(tasks)
	if candidates := AggregateRepairCandidates(tasks); len(candidates) != 1 || candidates[0].Parent.ID != ordinary.ID {
		t.Fatalf("ambiguous schedulable parent was not detected: %#v", candidates)
	}
	want := map[string]bool{ordinary.ID: true}
	for _, child := range ordinaryChildren {
		want[child.ID] = true
	}
	for _, ready := range frontier.Ready {
		delete(want, ready.ID)
	}
	if len(want) != 0 {
		t.Fatalf("descriptive hierarchy changed scheduling; missing ready IDs %#v", want)
	}
	_ = children
}

func TestAggregateCloseDerivesProgressAndRefusesUntilEveryChildVerifiedAndDone(t *testing.T) {
	w, parent, children := aggregateFixture(t)
	plan, _ := BuildAggregateRepairPlan(w, parent)
	if _, err := ApplyAggregateRepairPlan(w, parent, plan.ID); err != nil {
		t.Fatal(err)
	}
	parent, _ = FindTask(w, parent.ID)
	if err := CloseTask(w, parent, "a-root"); err == nil || !strings.Contains(err.Error(), "0/4") {
		t.Fatalf("aggregate closed over open children: %v", err)
	}

	for i, child := range children {
		child, _ = FindTask(w, child.ID)
		CheckAllAcceptance(child)
		if err := SaveTask(child); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if err := MoveTask(w, child, model.StatusBlocked); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := MoveTask(w, child, model.StatusDone); err != nil {
			t.Fatal(err)
		}
	}
	progress, err := AggregateProgressFor(w, parent)
	if err != nil || progress.RequiredDone != 3 || progress.ReadyToClose || !strings.Contains(strings.Join(progress.Blockers, " "), "blocked") {
		t.Fatalf("blocked progress = %#v err=%v", progress, err)
	}
	first, _ := FindTask(w, children[0].ID)
	if err := MoveTask(w, first, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	progress, err = AggregateProgressFor(w, parent)
	if err != nil || !progress.ReadyToClose || progress.RequiredDone != 4 {
		t.Fatalf("completed progress = %#v err=%v", progress, err)
	}
	parent, _ = FindTask(w, parent.ID)
	CheckAllAcceptance(parent)
	if err := CloseTask(w, parent, "a-root"); err != nil {
		t.Fatalf("fully complete aggregate refused: %v", err)
	}
}

func TestAggregateCloseHonorsPRLandingPolicy(t *testing.T) {
	w, parent, children := aggregateFixture(t)
	project, _ := LoadProject(w, "core")
	if err := ConfigureProjectLanding(project, model.LandingPolicy{Mode: model.LandingPR, Base: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveProject(project); err != nil {
		t.Fatal(err)
	}
	plan, _ := BuildAggregateRepairPlan(w, parent)
	if _, err := ApplyAggregateRepairPlan(w, parent, plan.ID); err != nil {
		t.Fatal(err)
	}
	for _, child := range children {
		child, _ = FindTask(w, child.ID)
		CheckAllAcceptance(child)
		if err := SaveTask(child); err != nil {
			t.Fatal(err)
		}
		if err := MoveTask(w, child, model.StatusDone); err != nil {
			t.Fatal(err)
		}
	}
	parent, _ = FindTask(w, parent.ID)
	progress, err := AggregateProgressFor(w, parent)
	if err != nil || progress.ReadyToClose || !strings.Contains(strings.Join(progress.Blockers, " "), "unlanded") {
		t.Fatalf("PR policy accepted unlanded children: %#v err=%v", progress, err)
	}
}

func TestAggregateRepairPlanRefusesGraphChangedAfterPreview(t *testing.T) {
	w, parent, _ := aggregateFixture(t)
	plan, err := BuildAggregateRepairPlan(w, parent)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTask(w, "a-root", "core", "late child", TaskOpts{Parent: parent.ID, Accept: []string{"late"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyAggregateRepairPlan(w, parent, plan.ID); err == nil || !strings.Contains(err.Error(), "plan changed") {
		t.Fatalf("stale plan apply = %v", err)
	}
	parent, _ = FindTask(w, parent.ID)
	if parent.IsAggregate() {
		t.Fatal("stale repair partially changed the parent")
	}
}

func TestOversizedLeafDecompositionIsStableAndOnlyApplyCreatesTasks(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "decomposition")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	parent, err := CreateTask(w, "a-root", "core", "Oversized delivery", TaskOpts{
		Estimate: "8,13,21",
		Accept:   []string{"internal/api/ serves requests", "internal/ui/ renders state", "docs/ explains operation"},
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := BuildDecompositionPlan(w, parent)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := BuildDecompositionPlan(w, parent)
	if first.ID != second.ID || len(first.Children) != 3 {
		t.Fatalf("proposal is not stable: %#v / %#v", first, second)
	}
	tasks, _ := ListTasks(w, "core", "")
	if len(tasks) != 1 {
		t.Fatalf("preview silently created %d tasks", len(tasks)-1)
	}
	for i, child := range first.Children {
		if len(child.Acceptance) != 1 || child.Estimate == "" || len(child.Claims) != 1 || (i > 0 && len(child.DependsOn) != 1) {
			t.Fatalf("child %d lacks executable WBS fields: %#v", i, child)
		}
	}
	if _, err := ApplyDecompositionPlan(w, parent, first.ID, "a-root"); err != nil {
		t.Fatal(err)
	}
	tasks, _ = ListTasks(w, "core", "")
	if len(tasks) != 4 {
		t.Fatalf("apply created %d total tasks, want 4", len(tasks))
	}
	for _, task := range tasks {
		if task.ID != parent.ID && len(task.Claims()) != 1 {
			t.Fatalf("applied child %s lost its minimal path claim: %#v", task.ID, task.Claims())
		}
	}
	parent, _ = FindTask(w, parent.ID)
	if !parent.IsAggregate() || len(parent.AggregateChildren()) != 3 {
		t.Fatalf("parent after apply = kind %s children %#v", parent.TaskKind(), parent.AggregateChildren())
	}
	if _, err := os.Stat(filepath.Join(w.Root, workspace.Dir, "plans", "decomposition", first.ID+".json")); err != nil {
		t.Fatalf("immutable plan missing: %v", err)
	}
}
