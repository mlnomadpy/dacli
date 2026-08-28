package ghmirror

import (
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestGitHubProjectionUsesOneParentIssueForDeliverySlices(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "delivery-projection")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	parent, err := store.CreateTask(w, "a-root", "core", "Product", store.TaskOpts{Accept: []string{"complete"}})
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := store.CreateDeliverySlice(w, "a-root", parent.ID, "API", []string{"API lands"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.CreateDeliverySlice(w, "a-root", parent.ID, "UI", []string{"UI lands"}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	whole, err := deliveryProjectionParents(w, []*store.Task{parent, first, second})
	if err != nil || len(whole) != 1 || whole[0].ID != parent.ID {
		t.Fatalf("whole project projection=%v err=%v", whole, err)
	}
	explicit, err := deliveryProjectionParents(w, []*store.Task{second})
	if err != nil || len(explicit) != 1 || explicit[0].ID != parent.ID {
		t.Fatalf("explicit slice projection=%v err=%v", explicit, err)
	}
}
