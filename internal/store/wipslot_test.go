package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// writeAgentFile creates a non-retired agent file in the given role, the way
// `agent spawn` does.
func writeAgentFile(t *testing.T, w *workspace.Workspace, id, role string) {
	t.Helper()
	d := &mdstore.Doc{}
	d.Front.Set("id", id)
	d.Front.Set("kind", "agent")
	d.Front.Set("role", role)
	d.Front.Set("grant", "rw")
	if err := mdstore.WriteFile(w.AgentPath(id), d); err != nil {
		t.Fatal(err)
	}
}

// writeFinishedRun records a run for `child` whose process is gone: PID 0 can
// never be live, which is what "ran and exited" looks like on disk.
func writeFinishedRun(t *testing.T, w *workspace.Workspace, runID, child string) {
	t.Helper()
	dir := w.RunDir(runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: runID, Child: child, PID: 0, PGID: 0}
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
}

func wipWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "wip")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// A WIP limit bounds CONCURRENT work. Counting every non-retired agent file
// made it bound LIFETIME work instead: nothing in the run lifecycle called
// RetireAgent, so each spawn's agent held its slot forever, roles filled up,
// and every later spawn was refused ("role fixer is at its WIP limit (3/3)")
// while `dacli agents` showed nobody live. The loop then no-opped for cycles,
// looking like progress and producing nothing (task 282, issues #382 / #418).
func TestFinishedAgentDoesNotHoldAWIPSlot(t *testing.T) {
	w := wipWS(t)
	for _, id := range []string{"a-fixer-aaa", "a-fixer-bbb", "a-fixer-ccc"} {
		writeAgentFile(t, w, id, "fixer")
		writeFinishedRun(t, w, "RUN"+id, id)
	}

	if got := ActiveInRole(w, "fixer"); got != 0 {
		t.Fatalf("ActiveInRole = %d; three agents that RAN and EXITED must hold no slots", got)
	}
}

// The discriminator cannot be "has a live process" alone. `agent spawn` mints
// an identity BEFORE any process exists — the token is handed to a child that
// runs afterwards, possibly outside dacli — so a just-minted agent is about to
// work and must keep its slot. Freeing it here would let a role oversubscribe
// itself, which is the opposite failure and just as real.
func TestJustMintedAgentStillHoldsItsSlot(t *testing.T) {
	w := wipWS(t)
	writeAgentFile(t, w, "a-fixer-new", "fixer")
	// No run record at all: minted, not yet started.

	if got := ActiveInRole(w, "fixer"); got != 1 {
		t.Fatalf("ActiveInRole = %d; an agent minted but never run is about to work and must hold its slot", got)
	}
}

// Retirement still frees a slot explicitly, and a different role's agents
// never count against this one — otherwise the count above could be right by
// accident.
func TestRetiredAndForeignAgentsAreNotCounted(t *testing.T) {
	w := wipWS(t)
	writeAgentFile(t, w, "a-fixer-live", "fixer")
	writeAgentFile(t, w, "a-reviewer-live", "reviewer")
	if got := ActiveInRole(w, "fixer"); got != 1 {
		t.Fatalf("ActiveInRole(fixer) = %d; another role's agent must not count", got)
	}
	if err := RetireAgent(w, "a-fixer-live"); err != nil {
		t.Fatal(err)
	}
	if got := ActiveInRole(w, "fixer"); got != 0 {
		t.Errorf("ActiveInRole after retire = %d; want 0", got)
	}
}
