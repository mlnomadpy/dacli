//go:build !windows

package procmon_test

import (
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

// makeZombie starts a child that exits immediately and deliberately does NOT
// Wait on it, reproducing the dacli-217 shape: a long-lived parent that
// released a detached child and never reaped it. The child is a zombie —
// exited, but still holding a process-table slot (and its PID) until its
// parent waits. Cleanup always waits, so no zombie outlives the test.
func makeZombie(t *testing.T) (pid, pgid int) {
	t.Helper()
	cmd := exec.Command("sh", "-c", "exit 0")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid = cmd.Process.Pid
	t.Cleanup(func() { _ = cmd.Wait() })

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st, ok := procmon.ProcState(pid)
		if ok && len(st) > 0 && st[0] == 'Z' {
			return pid, pid // Setpgid ⇒ leader pid == group id
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Skipf("could not observe pid %d in zombie state on this platform", pid)
	return 0, 0
}

// dacli-217: `syscall.Kill(pid, 0)` SUCCEEDS against a zombie, so a bare
// signal-0 probe reports an exited-but-unreaped child as live FOREVER inside a
// long-lived parent (`dacli mcp serve`). That is the phantom agent in
// `dacli agents`, the `dacli wait` that blocks to its timeout, and the KillTree
// that escalates to SIGKILL against a corpse. Liveness must exclude zombies.
func TestAliveRejectsAZombie(t *testing.T) {
	pid, pgid := makeZombie(t)

	// The premise: the cheap probe cannot tell the difference.
	if err := syscall.Kill(pid, 0); err != nil && err != syscall.EPERM {
		t.Skipf("signal-0 no longer reaches the zombie (%v) — premise gone", err)
	}
	if procmon.Alive(pid) {
		t.Error("Alive reported a zombie as live — dacli agents will show a phantom")
	}
	if procmon.AliveIdentity(pid, "") {
		t.Error("AliveIdentity reported a zombie as live")
	}
	if procmon.AliveRecord(procmon.Record{PID: pid, PGID: pgid}) {
		t.Error("AliveRecord reported a zombie as live")
	}
	// A group whose only member is a zombie is a dead group: `dacli wait` falls
	// back to GroupAlive when the leader looks gone, so a zombie-only group
	// would block the LAND step until the overall timeout.
	if procmon.GroupAlive(pgid) {
		t.Error("GroupAlive reported a zombie-only group as live")
	}
	// A zombie holds no memory and burns no CPU; counting it as a group member
	// would report a live agent's tree as bigger than it is.
	if u := procmon.SampleGroup(pgid); u.Procs != 0 {
		t.Errorf("SampleGroup counted %d zombie(s) as live process(es)", u.Procs)
	}
}

// The zombie exclusion must not cost us real liveness: a running process still
// reports alive through every probe. Without this, a bug in the state read
// (e.g. treating an unreadable state as dead) would silently make dacli blind
// to every live agent, which is worse than the phantom it fixes.
func TestAliveStillAcceptsARunningProcess(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	if !procmon.Alive(pid) {
		t.Error("a running process must be alive")
	}
	if !procmon.GroupAlive(pid) {
		t.Error("a running process's group must be alive")
	}
	if u := procmon.SampleGroup(pid); u.Procs < 1 {
		t.Errorf("SampleGroup saw %d proc in a live group", u.Procs)
	}
	if st, ok := procmon.ProcState(pid); !ok || len(st) == 0 || st[0] == 'Z' {
		t.Errorf("ProcState(%d) = %q, ok=%v; want a live (non-Z) state", pid, st, ok)
	}
}
