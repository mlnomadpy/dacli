package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

// A detached run can finish inside the startup grace used to bridge runtime
// registration races. Its durable terminal outcome must outrank that grace:
// otherwise agents reports nobody after the grace expires while an immediate
// follow-up spawn still sees the old path claim as live (issue #588).
func TestFinalizeRunAtomicallyReleasesClaimFromLivenessRecord(t *testing.T) {
	w := newExecWS(t)
	child, _, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "fixer", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	runID := runID(22)
	dir := w.RunDir(runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{
		RunID: runID, Child: child, Task: "t-finished", Role: "fixer",
		PID: 0, PGID: 0, Started: time.Now(), Claims: []string{"internal/features/execution"},
	}
	procPath := filepath.Join(dir, "proc.txt")
	if err := procmon.WriteRecord(procPath, rec); err != nil {
		t.Fatal(err)
	}

	finalizeRun(w, rec)
	// Explicit retirement after terminal finalization is a documented recovery
	// action and remains idempotent even though finalizeRun already retired it.
	if err := store.RetireAgent(w, child); err != nil {
		t.Fatal(err)
	}
	if err := store.RetireAgent(w, child); err != nil {
		t.Fatalf("second retirement was not idempotent: %v", err)
	}

	got, err := procmon.ReadRecord(procPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome == "" {
		t.Fatal("terminal outcome was not persisted in the liveness record")
	}
	if len(got.Claims) != 0 {
		t.Fatalf("terminal record retained claims %v", got.Claims)
	}
	live, err := liveAgents(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(live) != 0 {
		t.Fatalf("finalized run remained live during startup grace: %+v", live)
	}
}

// Recovery must not equate a stale file with a stale process. It releases only
// after the recorded identity is gone and both bounded activity windows have
// expired; a record for this live test process remains untouched.
func TestLiveAgentRecoveryReleasesOnlyIdentityProvenStaleClaims(t *testing.T) {
	w := newExecWS(t)
	staleID := runID(23)
	staleDir := w.RunDir(staleID)
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := procmon.Record{
		RunID: staleID, Child: "a-crashed", PID: 0, PGID: 0,
		Started: time.Now().Add(-runStartupGrace - time.Minute), Claims: []string{"internal/procmon"},
	}
	stalePath := filepath.Join(staleDir, "proc.txt")
	if err := procmon.WriteRecord(stalePath, stale); err != nil {
		t.Fatal(err)
	}
	live := writeLiveProcRecord(t, w, []string{"internal/features/execution"})

	gotLive, err := liveAgents(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotLive) != 1 || gotLive[0].RunID != live.RunID {
		t.Fatalf("liveAgents = %+v, want only %s", gotLive, live.RunID)
	}
	recovered, err := procmon.ReadRecord(stalePath)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Outcome == "" || len(recovered.Claims) != 0 {
		t.Fatalf("stale record was not terminally released: %+v", recovered)
	}
	stillLive, err := procmon.ReadRecord(filepath.Join(w.RunDir(live.RunID), "proc.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if stillLive.Outcome != "" || len(stillLive.Claims) != 1 {
		t.Fatalf("live record was released during recovery: %+v", stillLive)
	}
}
