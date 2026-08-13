package execution

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const timeoutMarker = "timed_out.txt"

// startRunWatchdog starts a separate dacli process because a goroutine dies
// with the `spawn --detach` launcher. The watchdog has no inherited pipes and
// therefore survives that launcher exiting.
func startRunWatchdog(root, runID string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "__run-watchdog", runID)
	cmd.Dir = root
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	setNewProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// RunWatchdog is the private subprocess entry point used by detached spawns.
func RunWatchdog(runID string) int {
	w, err := workspace.Find(".")
	if err != nil {
		return 1
	}
	id, err := agentid.Resolve(w)
	if err != nil || clikit.RequireRW(id, "enforcing a recorded runtime timeout") != nil {
		return 3
	}
	rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(runID), "proc.txt"))
	if err != nil || rec.RunID != runID || rec.Timeout <= 0 || rec.Started.IsZero() {
		return 1
	}
	return superviseRecordedRun(w, rec)
}

func superviseRecordedRun(w *workspace.Workspace, rec procmon.Record) int {
	if delay := time.Until(rec.Started.Add(rec.Timeout)); delay > 0 {
		time.Sleep(delay)
	}
	if !terminateRecordedTree(rec, 3*time.Second) {
		return 0
	}
	if err := markTimedOut(w, rec); err != nil {
		return 1
	}
	if rec.Child != "" {
		_ = store.RetireAgent(w, rec.Child)
	}
	recordExit(w, rec, "timed out", time.Since(rec.Started).Round(time.Second), fmt.Sprintf("exceeded recorded timeout %s; process tree terminated", rec.Timeout))
	return 0
}

// terminateRecordedTree requires a captured start identity immediately before
// signalling. Legacy/best-effort records may be observed, but never killed:
// an unverified numeric PID/PGID may now belong to an unrelated process.
func terminateRecordedTree(rec procmon.Record, grace time.Duration) bool {
	if rec.PIDStart == "" || !procmon.AliveIdentity(rec.PID, rec.PIDStart) {
		return false
	}
	termed, killed := procmon.KillTree(rec.PGID, grace)
	return termed || killed
}

func markTimedOut(w *workspace.Workspace, rec procmon.Record) error {
	runDir := w.RunDir(rec.RunID)
	elapsed := time.Since(rec.Started).Round(time.Second)
	marker := fmt.Sprintf("timed out after %s at %s\n", rec.Timeout, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(runDir, timeoutMarker), []byte(marker), 0o644); err != nil {
		return err
	}
	if err := procmon.CompleteRecord(filepath.Join(runDir, "proc.txt"), rec, "timed out"); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "outcome.md"), []byte(fmt.Sprintf(
		"outcome: timed out (detached)\nchild: %s\nelapsed_since_start: %s\ntimeout: %s\n", rec.Child, elapsed, rec.Timeout)), 0o644)
}
