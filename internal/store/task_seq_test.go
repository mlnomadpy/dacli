package store

import (
	"fmt"
	"sync"
	"testing"
)

// Two agents filing tasks at the same moment must never land on the same
// seq: a shared NNN makes both tasks unaddressable by number and trips
// FindTask's ambiguity check (dacli 209).
func TestCreateTaskConcurrentGetsDistinctSeqs(t *testing.T) {
	w := indexWorkspace(t)

	const n = 20
	var wg sync.WaitGroup
	results := make([]*Task, n)
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			task, err := CreateTask(w, "a-root", "core", fmt.Sprintf("concurrent task %d", i), TaskOpts{})
			results[i] = task
			errs[i] = err
		}(i)
	}
	close(start)
	wg.Wait()

	seen := map[int]string{}
	for i, task := range results {
		if errs[i] != nil {
			t.Fatalf("CreateTask[%d]: %v", i, errs[i])
		}
		if prev, ok := seen[task.Seq]; ok {
			t.Fatalf("seq %d allocated to both %q and %q", task.Seq, prev, task.Slug)
		}
		seen[task.Seq] = task.Slug
	}

	all, err := ListTasks(w, "core", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != n {
		t.Fatalf("expected %d tasks on disk, got %d (a seq collision let one file overwrite another)", n, len(all))
	}

	// FindTask must resolve every seq unambiguously.
	for seq, slug := range seen {
		got, err := FindTask(w, fmt.Sprintf("%d", seq))
		if err != nil {
			t.Fatalf("FindTask(%d): %v", seq, err)
		}
		if got.Slug != slug {
			t.Fatalf("FindTask(%d) = %q, want %q", seq, got.Slug, slug)
		}
	}
}
