//go:build windows

package procmon

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/commandresult"
)

// ProcState is the Windows stub of the unix scheduler-state read. Windows has
// no zombie state to detect: a terminated process is removed from tasklist
// immediately, and the un-waited-for handle a parent may still hold is not a
// process-table entry, so the dacli-217 phantom cannot arise here. Always
// ok=false, meaning "no evidence" — callers keep their tasklist verdict.
func ProcState(pid int) (string, bool) { return "", false }

// ProcStateChecked matches the Unix diagnostic-aware API. Windows has no
// scheduler-state subprocess to fail, so absence is ordinary no evidence.
func ProcStateChecked(pid int) (string, bool, error) { return "", false, nil }

// Alive reports whether pid names a live process. Windows has no signal-0
// probe (Process.Signal only supports os.Kill), so existence is checked via
// `tasklist`'s CSV output instead.
func Alive(pid int) bool {
	alive, _ := AliveChecked(pid)
	return alive
}

// AliveChecked exposes tasklist failure without changing Alive's best-effort
// bool contract.
func AliveChecked(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	cmd := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH", "/FO", "CSV")
	out, err := commandresult.Output(cmd, commandresult.RunOptions{Operation: "probe process liveness"})
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), "\""+strconv.Itoa(pid)+"\""), nil
}

// GroupAlive reports whether the group leader is still alive. Windows has no
// POSIX process group; PGID is always the leader's own PID (Setpgid is not
// set on this platform — see execution_windows.go), so this is Alive(pgid).
func GroupAlive(pgid int) bool {
	return Alive(pgid)
}

// GroupAliveChecked is the Windows diagnostic-aware group probe. A Windows
// process group is represented by its leader, so this delegates to tasklist.
func GroupAliveChecked(pgid int) (bool, error) { return AliveChecked(pgid) }

// KillTree terminates a whole process tree: `taskkill /T` first (closes
// windows / lets the tree exit cleanly), then `taskkill /T /F` after grace if
// anything survives — the Windows analogue of SIGTERM-then-SIGKILL. termed
// reports the first taskkill ran; killed reports the forceful one was needed.
func KillTree(pgid int, grace time.Duration) (termed, killed bool) {
	if pgid <= 0 {
		return false, false
	}
	if err := exec.Command("taskkill", "/T", "/PID", strconv.Itoa(pgid)).Run(); err == nil {
		termed = true
	}
	for waited := time.Duration(0); waited < grace; waited += 100 * time.Millisecond {
		if !GroupAlive(pgid) {
			return termed, false
		}
		time.Sleep(100 * time.Millisecond)
	}
	if GroupAlive(pgid) {
		if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pgid)).Run(); err == nil {
			killed = true
		}
	}
	return termed, killed
}
