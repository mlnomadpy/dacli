//go:build !windows

package execution

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

// A detached silent runtime with helpers is the failure shape from task 372:
// the launcher is gone, no transcript activity can wake a waiter, and only the
// persisted deadline can release its claim. The watchdog must kill the exact
// recorded group and finalize the outcome as timed out.
func TestRecordedTimeoutReapsSilentDetachedTreeAndFinalizes(t *testing.T) {
	w := newExecWS(t)
	cmd := exec.Command("sh", "-c", "sleep 60 & sleep 60 & wait")
	setNewProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	start, ok := procmon.ProcStart(pid)
	if !ok {
		_ = cmd.Process.Kill()
		t.Skip("cannot capture process start identity")
	}
	runID := runID(7)
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: runID, Child: "a-silent", PID: pid, PGID: pid, PIDStart: start,
		Started: time.Now().Add(-2 * time.Second), Timeout: time.Second, Claims: []string{"internal/store"}}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "outcome.md"), []byte(detachedRunningPlaceholder+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := superviseRecordedRun(w, rec); code != 0 {
		t.Fatalf("watchdog exit = %d", code)
	}
	_ = cmd.Wait()
	if procmon.GroupAlive(pid) {
		t.Fatalf("silent runtime group %d survived its recorded timeout", pid)
	}
	raw, err := os.ReadFile(filepath.Join(runDir, "outcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "outcome: timed out (detached)") {
		t.Fatalf("outcome did not record timeout:\n%s", raw)
	}
	_ = finalizeRun(w, rec) // a concurrently polling wait must preserve timeout
	raw, _ = os.ReadFile(filepath.Join(runDir, "outcome.md"))
	if !strings.Contains(string(raw), "outcome: timed out (detached)") {
		t.Fatalf("wait reconciliation overwrote the timeout verdict:\n%s", raw)
	}
	live, err := liveAgents(w)
	if err != nil || len(live) != 0 {
		t.Fatalf("claim remained live after timeout: live=%+v err=%v", live, err)
	}
}

func TestRecordedCleanupRefusesMissingOrMismatchedStartIdentity(t *testing.T) {
	pid := os.Getpid()
	if terminateRecordedTree(procmon.Record{PID: pid, PGID: pid}, 0) {
		t.Fatal("cleanup signalled a record with no PID start identity")
	}
	if terminateRecordedTree(procmon.Record{PID: pid, PGID: pid, PIDStart: "not this process"}, 0) {
		t.Fatal("cleanup signalled a recycled PID identity")
	}
}
