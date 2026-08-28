package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func deliveryWorkspace(t *testing.T) (*workspace.Workspace, *Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "delivery")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	parent, err := CreateTask(w, "a-root", "core", "Build product", TaskOpts{Accept: []string{"all product work lands"}})
	if err != nil {
		t.Fatal(err)
	}
	return w, parent
}

func TestDeliveryReconciliationResumesAcrossPRMergeAndCleanupPhases(t *testing.T) {
	w := gitWorkspace(t)
	parent, err := CreateTask(w, "a-root", "core", "Parent delivery", TaskOpts{Accept: []string{"all slices land"}})
	if err != nil {
		t.Fatal(err)
	}
	slice, _, err := CreateDeliverySlice(w, "a-root", parent.ID, "Restartable slice", []string{"verified"}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	branch := TaskBranch(slice)
	git(t, w.Root, "checkout", "-q", "-b", branch)
	if err := os.WriteFile(filepath.Join(w.Root, "delivery.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, w.Root, "add", "delivery.txt")
	git(t, w.Root, "commit", "-q", "-m", "delivery")
	headOut, err := gitx.Run(w.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	head := strings.TrimSpace(headOut)
	open := DeliveryPR{Number: 17, DeliveryConfidence: "OPEN", URL: "https://example.test/pull/17", HeadRefName: branch, HeadRefOid: head}
	if err := RecordDeliveryObservation(w, slice, open); err != nil {
		t.Fatal(err)
	}
	p, err := DeliveryProgressFor(w, parent)
	if err != nil {
		t.Fatal(err)
	}
	if p.Slices[0].Landed || p.Slices[0].PRNumber != 17 || p.Slices[0].HeadSHA != head || p.Slices[0].TreeSHA == "" {
		t.Fatalf("open/CI checkpoint=%#v", p.Slices[0])
	}
	merge := &struct {
		OID string `json:"oid"`
	}{OID: strings.Repeat("a", 40)}
	merged := open
	merged.DeliveryConfidence, merged.MergeCommit = "MERGED", merge
	if err := RecordDeliveryObservation(w, slice, merged); err != nil {
		t.Fatal(err)
	}
	if err := RecordDeliveryObservation(w, slice, merged); err != nil {
		t.Fatalf("restart replay duplicated/refused merged observation: %v", err)
	}
	p, err = DeliveryProgressFor(w, parent)
	if err != nil || !p.Slices[0].Landed || p.Slices[0].MergeSHA != merge.OID {
		t.Fatalf("merged checkpoint=%#v err=%v", p.Slices[0], err)
	}

	// A commit after the observed PR head is a new delivery generation in
	// substance and must invalidate the old merge observation.
	if err := os.WriteFile(filepath.Join(w.Root, "delivery.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, w.Root, "add", "delivery.txt")
	git(t, w.Root, "commit", "-q", "-m", "new unmerged work")
	p, err = DeliveryProgressFor(w, parent)
	if err != nil || p.Slices[0].Landed {
		t.Fatalf("older merged head satisfied newer work: %#v err=%v", p.Slices[0], err)
	}
	newHeadOut, err := gitx.Run(w.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	merged.Number, merged.URL, merged.HeadRefOid = 18, "https://example.test/pull/18", strings.TrimSpace(newHeadOut)
	merged.MergeCommit = &struct {
		OID string `json:"oid"`
	}{OID: strings.Repeat("b", 40)}
	if err := RecordDeliveryObservation(w, slice, merged); err != nil {
		t.Fatal(err)
	}
	slice, err = FindTask(w, slice.ID)
	if err != nil {
		t.Fatal(err)
	}
	slice.Doc.SetSection("Acceptance", "- [x] verified\n")
	if err := SaveTask(slice); err != nil {
		t.Fatal(err)
	}
	if err := MoveTask(w, slice, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	git(t, w.Root, "checkout", "-q", "main")
	git(t, w.Root, "branch", "-D", branch)
	p, err = DeliveryProgressFor(w, parent)
	if err != nil || !p.ReadyToClose || p.Slices[0].CleanupState != "cleaned" {
		t.Fatalf("post-cleanup recovery=%#v err=%v", p, err)
	}
	if err := CloseTask(w, parent, "a-root"); err != nil {
		t.Fatalf("parent reconciliation close: %v", err)
	}
	replayed, created, err := CreateDeliverySlice(w, "a-root", parent.ID, "Restartable slice", []string{"verified"}, true, true)
	if err != nil || created || replayed.ID != slice.ID {
		t.Fatalf("post-close restart duplicated slice: %#v created=%t err=%v", replayed, created, err)
	}
	if got, findErr := FindDeliverySlice(w, fmt.Sprintf("%s/g%d", parent.ID, slice.DeliveryGeneration())); findErr != nil || got.ID != slice.ID {
		t.Fatalf("parent/generation lookup=%v err=%v", got, findErr)
	}
}

func TestDeliverySlicesAreTypedChildrenWithIndependentGenerationBranches(t *testing.T) {
	w, parent := deliveryWorkspace(t)
	first, created, err := CreateDeliverySlice(w, "a-root", parent.ID, "API slice", []string{"API works"}, true, false)
	if err != nil || !created {
		t.Fatalf("first slice: created=%t err=%v", created, err)
	}
	second, created, err := CreateDeliverySlice(w, "a-root", parent.ID, "UI slice", []string{"UI works"}, true, true)
	if err != nil || !created {
		t.Fatalf("second slice: created=%t err=%v", created, err)
	}
	if !first.IsDeliverySlice() || first.ParentID() != parent.ID || first.DeliveryGeneration() != 1 || second.DeliveryGeneration() != 2 {
		t.Fatalf("slice identities: first=%#v second=%#v", first, second)
	}
	if TaskBranch(first) == TaskBranch(second) || !strings.Contains(TaskBranch(first), "slice-g1-r0") || !strings.Contains(TaskBranch(second), "slice-g2-r0") {
		t.Fatalf("branches are not generation scoped: %q %q", TaskBranch(first), TaskBranch(second))
	}
	freshParent, err := FindTask(w, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(freshParent.Deps()) != 2 {
		t.Fatalf("required slices not projected into dependency/critical-path graph: %#v", freshParent.Deps())
	}

	// Restart/replay of the same requested slice is idempotent. Removing this
	// title match creates a duplicate child and makes the assertion fail.
	replayed, created, err := CreateDeliverySlice(w, "a-root", parent.ID, "API slice", []string{"API works"}, true, false)
	if err != nil || created || replayed.ID != first.ID {
		t.Fatalf("replay created duplicate: %#v created=%t err=%v", replayed, created, err)
	}
}

func TestHistoricalMergedSlicesCannotSatisfyNewParentGeneration(t *testing.T) {
	w, parent := deliveryWorkspace(t)
	old, _, err := CreateDeliverySlice(w, "a-root", parent.ID, "First attempt", []string{"lands"}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	old.Doc.SetSection("Acceptance", "- [x] lands\n")
	old.Doc.Front.Set("delivery_head_sha", "old-head")
	old.Doc.Front.Set("delivery_merge_sha", "old-merge")
	old.Doc.Front.Set("delivery_tree_sha", "old-tree")
	old.Doc.Front.Set("delivery_observed_at", "2026-01-01T00:00:00Z")
	if err := SaveTask(old); err != nil {
		t.Fatal(err)
	}
	if err := MoveTask(w, old, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	parent.Doc.Front.Set("generation", "1")
	if err := SaveTask(parent); err != nil {
		t.Fatal(err)
	}

	p, err := DeliveryProgressFor(w, parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Slices) != 0 || p.RequiredDone != 0 {
		t.Fatalf("historical generation leaked into progress: %#v", p)
	}
	newer, _, err := CreateDeliverySlice(w, "a-root", parent.ID, "Corrective attempt", []string{"new work lands"}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if newer.DeliveryParentGeneration() != 1 || TaskBranch(newer) == TaskBranch(old) {
		t.Fatalf("new generation identity aliased old slice: old=%q new=%q", TaskBranch(old), TaskBranch(newer))
	}
	if err := RefuseIncompleteDelivery(w, parent); err == nil {
		t.Fatal("open corrective slice did not block parent closure")
	}
}

func TestParentCloseRefusesPartialSliceAndAllowsLegacyOnePRTask(t *testing.T) {
	w, parent := deliveryWorkspace(t)
	if err := RefuseIncompleteDelivery(w, parent); err != nil {
		t.Fatalf("legacy one-PR task lost compatibility: %v", err)
	}
	if _, _, err := CreateDeliverySlice(w, "a-root", parent.ID, "Required slice", []string{"verified"}, true, false); err != nil {
		t.Fatal(err)
	}
	if err := CloseTask(w, parent, "a-root"); err == nil || !strings.Contains(err.Error(), "0/1 ready") {
		t.Fatalf("partial parent close error=%v", err)
	}
	if parent.Status == model.StatusDone {
		t.Fatal("partial close moved parent to done")
	}
}
