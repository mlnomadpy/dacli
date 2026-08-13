package execution

import (
	"os"
	"path/filepath"
	"strings"
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

// A status read may recover a dead detached run before wait sees it. The
// terminal outcome in proc.txt is the shared lifecycle state between those
// commands: it must outrank the still-running outcome.md placeholder and its
// freshly-created transcript, or wait treats the recovered corpse as active
// until transcript grace expires (task 436).
func TestAgentsRecoveryLetsWaitFinalizeMultipleNamedRuns(t *testing.T) {
	w := newExecWS(t)
	claims := [][]string{
		{"internal/features/execution"},
		{"internal/procmon"},
	}
	for i, paths := range claims {
		id := runID(24 + i)
		child := "a-" + id
		dir := mkRun(t, w, id, detachedRunningPlaceholder+"\nchild: "+child+"\ntask: t-recovered\n")
		if err := os.WriteFile(filepath.Join(dir, "transcript.log"), []byte("worker stopped\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		stale := time.Now().Add(-transcriptActiveGrace - time.Minute)
		if err := os.Chtimes(filepath.Join(dir, "transcript.log"), stale, stale); err != nil {
			t.Fatal(err)
		}
		if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
			RunID: id, Child: child, Task: "t-recovered",
			PID: 0, PGID: 0, Started: time.Now().Add(-runStartupGrace - time.Minute), Claims: paths,
		}); err != nil {
			t.Fatal(err)
		}
	}

	ctx, _, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	for i := range claims {
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(runID(24+i)), "proc.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if rec.Outcome != "process exited (recovered)" {
			t.Fatalf("run %s outcome = %q, want recovered terminal state", rec.RunID, rec.Outcome)
		}
		if len(rec.Claims) != 0 {
			t.Fatalf("run %s retained recovered claims %v", rec.RunID, rec.Claims)
		}
		// A detached writer can advance after the guardian was recovered. This
		// liveness hint must not resurrect the terminal proc.txt record.
		now := time.Now()
		if err := os.Chtimes(filepath.Join(w.RunDir(rec.RunID), "transcript.log"), now, now); err != nil {
			t.Fatal(err)
		}
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdWait(ctx, []string{runID(24), runID(25), "--interval", "1", "--timeout", "1"}); err != nil {
		t.Fatalf("wait did not consume agents' recovered terminal state: %v\n%s", err, out)
	}
	if got := strings.Count(out.String(), "no visible result"); got != 2 {
		t.Fatalf("wait finalized %d recovered run(s), want 2:\n%s", got, out)
	}
	for i := range claims {
		raw, err := os.ReadFile(filepath.Join(w.RunDir(runID(24+i)), "outcome.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "outcome: no visible result (detached)") {
			t.Fatalf("run %s outcome.md was not finalized:\n%s", runID(24+i), raw)
		}
	}
}
