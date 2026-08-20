package store

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
)

func TestTaskSnapshotRefusesInvalidStateAndRefreshSeesTransition(t *testing.T) {
	w := indexWorkspace(t)
	task, err := CreateTask(w, "a-root", "core", "snapshot boundary", TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := LoadTaskSnapshot(w)
	if err != nil {
		t.Fatal(err)
	}
	if err := MoveTask(w, task, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	s.Invalidate()
	if _, err := s.Find(task.ID); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("invalid snapshot lookup = %v, want explicit refusal", err)
	}
	if err := s.Refresh(); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.Find(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Status != model.StatusDone {
		t.Fatalf("refreshed status = %s, want done", fresh.Status)
	}
}

func TestIndexedTenRefResolutionUsesMateriallyFewerAllocations(t *testing.T) {
	w := indexWorkspace(t)
	refs := make([]string, 0, 10)
	for i := 0; i < 100; i++ {
		task, err := CreateTask(w, "a-root", "core", fmt.Sprintf("lookup fixture %03d", i), TaskOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if i < 10 {
			refs = append(refs, task.ID)
		}
	}
	walkAllocs := testing.AllocsPerRun(3, func() {
		for _, ref := range refs {
			if _, err := FindTask(w, ref); err != nil {
				panic(err)
			}
		}
	})
	indexedAllocs := testing.AllocsPerRun(3, func() {
		idx, err := BuildTaskIndex(w)
		if err != nil {
			panic(err)
		}
		for _, ref := range refs {
			if _, err := idx.Find(ref); err != nil {
				panic(err)
			}
		}
	})
	if indexedAllocs*3 >= walkAllocs {
		t.Fatalf("indexed 10-ref resolution allocations = %.0f, ten walks = %.0f; want indexed path at least 3x lower", indexedAllocs, walkAllocs)
	}
}
