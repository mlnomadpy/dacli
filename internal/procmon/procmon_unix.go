//go:build !windows

package procmon

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mlnomadpy/dacli/internal/commandresult"
)

// ProcState returns pid's OS scheduler state letter ("R", "S", "Z", ...).
// Linux answers from /proc (a cheap file read); everything else — macOS in
// particular — falls back to `ps -o state=`. ok=false means the state could
// not be read at all, which callers must treat as "no evidence", never as
// "dead": being unable to see a process is not the same as it having exited.
func ProcState(pid int) (string, bool) {
	state, ok, _ := ProcStateChecked(pid)
	return state, ok
}

// ProcStateChecked preserves ProcState's no-evidence answer while allowing a
// diagnostic-aware caller to distinguish an unreadable OS probe from an
// absent process. Linux's direct /proc miss still falls through to ps because
// containers and proc mounts vary.
func ProcStateChecked(pid int) (string, bool, error) {
	if pid <= 0 {
		return "", false, nil
	}
	// Linux: /proc/<pid>/stat. The comm field is parenthesised and may itself
	// contain spaces and parens, so the state letter is the first field AFTER
	// the last ')'.
	if raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		s := string(raw)
		if i := strings.LastIndex(s, ")"); i >= 0 {
			if f := strings.Fields(s[i+1:]); len(f) > 0 {
				return f[0], true, nil
			}
		}
		return "", false, nil
	}
	cmd := exec.Command("ps", "-o", "state=", "-p", strconv.Itoa(pid))
	out, err := commandresult.Output(cmd, commandresult.RunOptions{Operation: "probe process state"})
	if err != nil {
		return "", false, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", false, nil
	}
	return s, true, nil
}

// zombie reports whether pid has EXITED but not yet been waited for by its
// parent (dacli 217). This matters because dacli's detach path hands a child
// off without ever calling Wait: in a short-lived parent (`dacli spawn` from a
// shell) init reaps it and nobody notices, but in a long-lived parent
// (`dacli mcp serve`, or any in-process driver) the corpse lingers — and
// signal-0 SUCCEEDS against a corpse. Without this check every liveness probe
// would call the exited agent live forever.
func zombie(pid int) bool {
	st, ok := ProcState(pid)
	if !ok {
		return false // no evidence — leave the signal-0 verdict alone
	}
	return strings.HasPrefix(st, "Z")
}

// Alive reports whether pid names a live process, via a signal-0 probe (send
// no signal, just test existence/permission). EPERM means it exists but is
// owned elsewhere — still alive for our purposes. A zombie is excluded: it
// answers signal-0 but has already exited (dacli 217), and reporting it live
// gives `dacli agents` a phantom agent and `dacli wait` a hang.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err != nil && err != syscall.EPERM {
		return false
	}
	return !zombie(pid)
}

// GroupAlive reports whether ANY member of the process group still exists.
// This is the runaway check: the leader can exit while forked children keep
// running, and the group lives as long as one member does. Zombies do not
// count as members (dacli 217): a group whose survivors have all exited but
// gone unreaped is dead, and treating it as live makes KillTree escalate to
// SIGKILL against corpses and `dacli wait` block until its timeout.
func GroupAlive(pgid int) bool {
	live, err := GroupAliveChecked(pgid)
	if err != nil {
		return true // unreadable probe is not evidence that the group exited
	}
	return live
}

// GroupAliveChecked exposes a failed process-table read while GroupAlive keeps
// its safety-biased answer. This matters for kill/reconciliation callers: an
// unreadable ps must not authorize treating a possibly-live tree as dead.
func GroupAliveChecked(pgid int) (bool, error) {
	if pgid <= 0 {
		return false, nil
	}
	err := syscall.Kill(-pgid, 0)
	if err != nil && err != syscall.EPERM {
		return false, nil
	}
	// The group exists; check that at least one member is more than a corpse.
	// Only reachable when the cheap probe already said yes, so the ps cost is
	// paid on live-looking groups, not on every dead one.
	live, ok, probeErr := groupHasRunningMemberChecked(pgid)
	if probeErr != nil {
		return false, probeErr
	}
	if !ok {
		return true, nil
	}
	return live, nil
}

func groupHasRunningMemberChecked(pgid int) (bool, bool, error) {
	cmd := exec.Command("ps", "-A", "-o", "pgid=,state=")
	out, err := commandresult.Output(cmd, commandresult.RunOptions{Operation: "probe process group"})
	if err != nil {
		return false, false, err
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		if pg, _ := strconv.Atoi(fields[0]); pg != pgid {
			continue
		}
		if !strings.HasPrefix(fields[1], "Z") {
			return true, true, nil
		}
	}
	return false, true, nil
}

// KillTree terminates a whole process group: SIGTERM first, so the tree gets a
// chance to flush and exit cleanly, then SIGKILL after grace if anything
// survives. Signalling the NEGATIVE pgid reaches EVERY member — the agent AND
// every subprocess it forked — which is the entire point: no orphaned runaways
// left holding RAM/CPU/GPU. termed reports the SIGTERM landed; killed reports a
// SIGKILL was needed.
func KillTree(pgid int, grace time.Duration) (termed, killed bool) {
	if pgid <= 0 {
		return false, false
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err == nil {
		termed = true
	}
	for waited := time.Duration(0); waited < grace; waited += 100 * time.Millisecond {
		if !GroupAlive(pgid) {
			return termed, false
		}
		time.Sleep(100 * time.Millisecond)
	}
	if GroupAlive(pgid) {
		if err := syscall.Kill(-pgid, syscall.SIGKILL); err == nil {
			killed = true
		}
	}
	return termed, killed
}
