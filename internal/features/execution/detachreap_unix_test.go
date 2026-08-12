//go:build !windows

package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

// dacli-217: a detached child must not be left as a zombie inside a LONG-LIVED
// parent. `dacli spawn` from a shell exits straight away and init reaps the
// child, which is why this never showed up; `dacli mcp serve` (and this test
// binary, and any in-process driver) lives on, so an unreaped child keeps its
// process-table slot and its PID indefinitely. This asserts the parent-side
// reaper: after the child exits, its PID leaves the process table entirely —
// not merely "reports dead", which the zombie-aware liveness alone would give.
func TestDetachedChildIsReapedByALongLivedParent(t *testing.T) {
	if _, ok := procmon.ProcState(os.Getpid()); !ok {
		t.Skip("requires process-table visibility to distinguish a reaped child from an unreadable PID")
	}
	bin, _ := recorderBinary(t, "exit 0\n")
	rt := store.Runtime{Binary: bin, Mode: "stdin"}

	var pid int
	if _, _, err := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "t.log"), rt,
		"brief", "tok", nil, 30, true, func(p, _ int) { pid = p }); err != nil {
		t.Fatal(err)
	}
	if pid <= 0 {
		t.Fatalf("detached spawn reported pid %d", pid)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := procmon.ProcState(pid); !ok {
			return // fully reaped: the slot is gone
		}
		time.Sleep(50 * time.Millisecond)
	}
	st, _ := procmon.ProcState(pid)
	t.Fatalf("detached child pid %d still in the process table as %q after exiting — "+
		"a long-lived parent leaked a zombie", pid, st)
}

// The reaper must not make a detached child die with, or wait on, its parent:
// detach exists so the agent outlives this process. A child that is still
// running some seconds later must still be running, and still be reported live.
func TestDetachedChildKeepsRunningAfterTheParentReturns(t *testing.T) {
	if _, ok := procmon.ProcState(os.Getpid()); !ok {
		t.Skip("requires process-table visibility to observe detached-child liveness")
	}
	bin, _ := recorderBinary(t, "sleep 30\n")
	rt := store.Runtime{Binary: bin, Mode: "stdin"}

	var pid, pgid int
	start := time.Now()
	elapsed, timedOut, err := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "t.log"), rt,
		"brief", "tok", nil, 30, true, func(p, g int) { pid, pgid = p, g })
	if err != nil || timedOut || elapsed != 0 {
		t.Fatalf("detached start = (%v, %v, %v); it must return immediately", elapsed, timedOut, err)
	}
	if wall := time.Since(start); wall > 5*time.Second {
		t.Fatalf("detached start blocked for %s — the reaper must not Wait inline", wall)
	}
	// Kill the tree AND let the pid leave the process table before returning:
	// t.TempDir cleanup runs after this defer, and a child still holding the
	// recorder's capture dir races RemoveAll — the same flake shape fixed in
	// TestExecRuntimeDetachedReportsPID.
	defer func() {
		procmon.KillTree(pgid, time.Second)
		awaitObservableExit(t, pid)
	}()

	time.Sleep(500 * time.Millisecond)
	if !procmon.Alive(pid) {
		t.Error("a still-running detached child must report alive")
	}
	if !procmon.GroupAlive(pgid) {
		t.Error("a still-running detached child's group must report alive")
	}
}

// awaitObservableExit is only for tests that established process-table
// visibility before spawning. A deliberately killed recorder cannot write its
// normal completion marker, so its observable disappearance is the correct
// cleanup signal here; callers without that premise must use the filesystem
// completion marker instead.
func awaitObservableExit(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := procmon.ProcState(pid); !ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("detached child pid %d remained observable after it was killed", pid)
}
