package slices

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func sliceEnv(t *testing.T) (*workspace.Workspace, *store.Task, *clikit.Ctx) {
	t.Helper()
	prior, had := os.LookupEnv("DACLI_AGENT")
	if err := os.Unsetenv("DACLI_AGENT"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("DACLI_AGENT", prior)
		} else {
			_ = os.Unsetenv("DACLI_AGENT")
		}
	})
	w, err := workspace.Init(t.TempDir(), "slices")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	parent, err := store.CreateTask(w, "a-root", "core", "Parent", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	return w, parent, &clikit.Ctx{Cwd: w.Root, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
}

func TestSliceAddAndProgressJSONShareTypedState(t *testing.T) {
	_, parent, ctx := sliceEnv(t)
	if err := cmdSliceAdd(ctx, []string{"--task", parent.ID, "--title", "API delivery", "--accept", "API verified", "--terminal"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "slice created") {
		t.Fatalf("add output=%q", ctx.Stdout.(*bytes.Buffer).String())
	}
	ctx.Stdout = &bytes.Buffer{}
	ctx.JSON = true
	if err := cmdTaskProgress(ctx, []string{parent.ID}); err != nil {
		t.Fatal(err)
	}
	var got store.DeliveryProgress
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != "delivery-progress/v1" || len(got.Slices) != 1 || !got.Slices[0].Terminal || got.RequiredTotal != 1 || got.ReadyToClose {
		t.Fatalf("progress=%#v", got)
	}
}

func TestTaskProgressUsesAggregateDerivedState(t *testing.T) {
	w, parent, ctx := sliceEnv(t)
	for _, title := range []string{"API", "UI"} {
		if _, err := store.CreateTask(w, "a-root", "core", title, store.TaskOpts{Parent: parent.ID, Accept: []string{title}}); err != nil {
			t.Fatal(err)
		}
	}
	plan, _ := store.BuildAggregateRepairPlan(w, parent)
	if _, err := store.ApplyAggregateRepairPlan(w, parent, plan.ID); err != nil {
		t.Fatal(err)
	}
	ctx.JSON = true
	if err := cmdTaskProgress(ctx, []string{parent.ID}); err != nil {
		t.Fatal(err)
	}
	var got store.AggregateProgress
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != "aggregate-progress/v1" || got.Required != 2 || got.RequiredDone != 0 || got.ReadyToClose {
		t.Fatalf("aggregate progress=%#v", got)
	}
}

func TestSliceReconcileChoosesNewestExactBranchPRAndClearsHistoricalMerge(t *testing.T) {
	w, parent, ctx := sliceEnv(t)
	slice, _, err := store.CreateDeliverySlice(w, "a-root", parent.ID, "API delivery", []string{"API verified"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	branch := store.TaskBranch(slice)
	merge := &struct {
		OID string `json:"oid"`
	}{OID: "old-merge"}
	original := store.ObserveDeliveryPRs
	t.Cleanup(func() { store.ObserveDeliveryPRs = original })
	store.ObserveDeliveryPRs = func(string) ([]store.DeliveryPR, error) {
		return []store.DeliveryPR{
			{Number: 4, DeliveryConfidence: "MERGED", URL: "https://example/pull/4", HeadRefName: branch, HeadRefOid: "old-head", MergeCommit: merge},
			{Number: 9, DeliveryConfidence: "OPEN", URL: "https://example/pull/9", HeadRefName: branch, HeadRefOid: "new-head"},
		}, nil
	}
	ctx.JSON = true
	if err := cmdSliceReconcile(ctx, []string{"--task", parent.ID}); err != nil {
		t.Fatal(err)
	}
	fresh, err := store.FindTask(w, slice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := fresh.Doc.Front.Get("delivery_pr_number"); got != "9" {
		t.Fatalf("canonical PR=%q, want newest 9", got)
	}
	if got, _ := fresh.Doc.Front.Get("delivery_head_sha"); got != "new-head" {
		t.Fatalf("head=%q", got)
	}
	if got, ok := fresh.Doc.Front.Get("delivery_merge_sha"); ok || got != "" {
		t.Fatalf("older merged PR survived newer open generation: %q", got)
	}
}
