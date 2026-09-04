package eventlog

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func redundantPREventFixture(t *testing.T, count int) (*workspace.Workspace, *store.Task, string) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "redundant-pr")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "p", "landed", store.TaskOpts{Accept: []string{"verified"}})
	if err != nil {
		t.Fatal(err)
	}
	store.CheckAllAcceptance(task)
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, task, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	url := "https://github.com/acme/widgets/pull/44"
	for i := 0; i < count; i++ {
		if _, err := Append(w, "a-worker", model.EventComment, task.ID, "", "PR opened: "+url); err != nil {
			t.Fatal(err)
		}
	}
	return w, task, url
}

func TestRedundantPREventPlanDismissesFourWithExactMergeEvidence(t *testing.T) {
	w, task, url := redundantPREventFixture(t, 4)
	now := time.Unix(100, 0).UTC()
	obs := MergedPRObservation{Number: 44, URL: url, Head: store.TaskBranch(task), HeadOID: strings.Repeat("a", 40), MergeCommit: strings.Repeat("b", 40), Source: "github:gh-pr-list"}
	plan, err := PlanRedundantPREvents(w, "p", []MergedPRObservation{obs}, now)
	if err != nil || len(plan.Items) != 4 || len(plan.ID) != 64 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	if err := ApplyRedundantPRPlan(w, "a-root", plan); err != nil {
		t.Fatal(err)
	}
	pending, err := List(w, Query{Pending: true})
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending=%d err=%v", len(pending), err)
	}
	dismissals, _ := List(w, Query{Kinds: []model.EventKind{model.EventDismissal}})
	if len(dismissals) != 4 {
		t.Fatalf("dismissals=%d, want 4", len(dismissals))
	}
	for _, event := range dismissals {
		for _, want := range []string{plan.ID, task.ID, url, obs.MergeCommit, now.Format(time.RFC3339), obs.Source} {
			if !strings.Contains(event.Body, want) {
				t.Fatalf("dismissal omitted %q: %s", want, event.Body)
			}
		}
	}
	if _, err := os.Stat(workspaceEventPRSnapshot(w, plan.ID)); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if err := ApplyRedundantPRPlan(w, "a-root", plan); err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	after, _ := List(w, Query{Kinds: []model.EventKind{model.EventDismissal}})
	if len(after) != 4 {
		t.Fatalf("replay duplicated dismissals: %d", len(after))
	}
}

func TestRedundantPREventPlanFailsClosedOnStaleOrIncompleteEvidence(t *testing.T) {
	w, task, url := redundantPREventFixture(t, 1)
	valid := MergedPRObservation{Number: 44, URL: url, Head: store.TaskBranch(task), HeadOID: "head", MergeCommit: "merge", Source: "github"}
	for _, tc := range []struct {
		name string
		obs  MergedPRObservation
	}{
		{name: "stale URL", obs: MergedPRObservation{Number: 45, URL: url + "-old", Head: valid.Head, HeadOID: "head", MergeCommit: "merge", Source: "github"}},
		{name: "wrong branch", obs: MergedPRObservation{Number: 44, URL: url, Head: "dacli/999-other", HeadOID: "head", MergeCommit: "merge", Source: "github"}},
		{name: "missing merge commit", obs: MergedPRObservation{Number: 44, URL: url, Head: valid.Head, HeadOID: "head", Source: "github"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := PlanRedundantPREvents(w, "p", []MergedPRObservation{tc.obs}, time.Now())
			if err != nil || len(plan.Items) != 0 {
				t.Fatalf("unsafe candidate selected: %+v err=%v", plan.Items, err)
			}
		})
	}
	validPlan, err := PlanRedundantPREvents(w, "p", []MergedPRObservation{valid}, time.Now())
	if err != nil || len(validPlan.Items) != 1 {
		t.Fatalf("valid plan=%+v err=%v", validPlan, err)
	}
	task.Doc.Front.Set("generation", "1")
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanRedundantPREvents(w, "p", []MergedPRObservation{valid}, time.Now())
	if err != nil || len(plan.Items) != 0 {
		t.Fatalf("reopened generation selected: %+v err=%v", plan.Items, err)
	}
	if err := ApplyRedundantPRPlan(w, "a-root", validPlan); err == nil || !strings.Contains(err.Error(), "became stale") {
		t.Fatalf("stale apply = %v", err)
	}
}

func workspaceEventPRSnapshot(w *workspace.Workspace, planID string) string {
	return w.Root + "/.dacli/event-pr-reconciliation/" + planID + "/snapshot.json"
}
