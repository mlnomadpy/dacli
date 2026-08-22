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

	if got, err := ActiveInRole(w, "fixer"); err != nil || got != 0 {
		t.Fatalf("ActiveInRole = (%d, %v); three agents that RAN and EXITED must hold no slots", got, err)
	}
}

// Minting is identity history, not execution occupancy. Role removal still
// treats this state conservatively (remove_test.go), but a mature workspace's
// never-started identities must not make the live WIP roster look saturated.
func TestNeverStartedAgentDoesNotConsumeLiveOccupancy(t *testing.T) {
	w := wipWS(t)
	writeAgentFile(t, w, "a-fixer-new", "fixer")

	if got, err := ActiveInRole(w, "fixer"); err != nil || got != 0 {
		t.Fatalf("ActiveInRole = (%d, %v); never-started identity must consume no live occupancy", got, err)
	}
}

// Retirement changes identity provenance, not process truth. A live run still
// occupies its recorded role, while another role's run never counts here.
func TestLiveOccupancyUsesRunRoleDespiteRetirement(t *testing.T) {
	w := wipWS(t)
	writeAgentFile(t, w, "a-fixer-live", "fixer")
	writeAgentFile(t, w, "a-reviewer-live", "reviewer")
	pid := os.Getpid()
	start, _ := procmon.ProcStart(pid)
	for runID, child := range map[string]string{"RUN-FIXER": "a-fixer-live", "RUN-REVIEWER": "a-reviewer-live"} {
		dir := w.RunDir(runID)
		role := "fixer"
		if child == "a-reviewer-live" {
			role = "reviewer"
		}
		if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{RunID: runID, Child: child, Role: role, PID: pid, PGID: pid, PIDStart: start}); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := ActiveInRole(w, "fixer"); err != nil || got != 1 {
		t.Fatalf("ActiveInRole(fixer) = (%d, %v); another role's agent must not count", got, err)
	}
	if err := RetireAgent(w, "a-fixer-live"); err != nil {
		t.Fatal(err)
	}
	if got, err := ActiveInRole(w, "fixer"); err != nil || got != 1 {
		t.Errorf("ActiveInRole after retiring a live identity = (%d, %v); want (1, nil)", got, err)
	}
}

// makeAgentsDirUnreadable replaces the agents directory with a regular file
// so os.ReadDir(w.AgentsDir()) fails with a non-ENOENT error (ENOTDIR) rather
// than "no agents yet" — the transient-fault shape that must not be confused
// with an empty roster (mirrors internal/features/execution's
// makeRunsDirUnreadable, dacli 337's technique applied to the agents dir).
func makeAgentsDirUnreadable(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	if err := os.RemoveAll(w.AgentsDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(w.AgentsDir()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(w.AgentsDir(), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// dacli 341: an unreadable agents dir is a real fault, not zero agents.
// ActiveInRole used to swallow ListAgents' error and return 0, which let
// gateRoleWIP read "I could not check the WIP cap" as "the cap has nobody
// against it" and wave a spawn straight through — the 337 class ("a gate
// must never certify what it could not read") on gateRoleWIP's sibling gate.
func TestActiveInRoleFailsOnUnreadableAgentsDir(t *testing.T) {
	w := wipWS(t)
	makeAgentsDirUnreadable(t, w)

	if got, err := ActiveInRole(w, "fixer"); err == nil {
		t.Fatalf("ActiveInRole on an unreadable agents dir = (%d, nil), want a non-nil error", got)
	}
}
