package execution

import (
	"strings"
	"sync"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

// A respawn under the same role must bind the task to the identity minted for
// that spawn, never to a retired roster entry left in durable history (#725).
func TestClaimTaskAppendsNewlyMintedChildAfterRetiredSameRoleIdentity(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "identity transaction", store.TaskOpts{})
	parent := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}
	stale, _, err := agentid.Spawn(w, parent, "fixer", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	claimTask(ctx, w, task, stale)
	if err := store.RetireAgent(w, stale); err != nil {
		t.Fatal(err)
	}
	current, _, err := agentid.Spawn(w, parent, "fixer", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	claimTask(ctx, w, task, current)
	claimTask(ctx, w, task, current) // same-child retry is idempotent

	got, err := store.FindTask(w, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if claimant := store.ClaimedBy(got); claimant != current {
		t.Fatalf("latest claimant = %q, want newly minted child %q (retired %q)", claimant, current, stale)
	}
	log, _ := got.Doc.Section("Log")
	if strings.Count(log.Content, "claimed by "+current) != 1 {
		t.Fatalf("same child was not idempotent or was omitted:\n%s", log.Content)
	}
}

// WithTask serialization must not collapse concurrent same-role identities
// into whichever historical agent happens to be found first.
func TestClaimTaskKeepsDistinctConcurrentSameRoleChildren(t *testing.T) {
	w := newExecWS(t)
	parent := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}
	tasks := []*store.Task{
		mustTask(t, w, "parallel A", store.TaskOpts{}),
		mustTask(t, w, "parallel B", store.TaskOpts{}),
	}
	children := make([]string, 2)
	for i := range children {
		child, _, err := agentid.Spawn(w, parent, "fixer", model.GrantRW)
		if err != nil {
			t.Fatal(err)
		}
		children[i] = child
	}
	if children[0] == children[1] {
		t.Fatal("same-role spawns minted the same identity")
	}

	var wg sync.WaitGroup
	for i := range tasks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, _, _ := newCtx(w.Root)
			claimTask(ctx, w, tasks[i], children[i])
		}(i)
	}
	wg.Wait()
	for i := range tasks {
		got, err := store.FindTask(w, tasks[i].ID)
		if err != nil {
			t.Fatal(err)
		}
		if claimant := store.ClaimedBy(got); claimant != children[i] {
			t.Errorf("task %d claimant = %q, want its minted child %q", i, claimant, children[i])
		}
	}
}
