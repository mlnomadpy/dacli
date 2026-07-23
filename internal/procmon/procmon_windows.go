//go:build windows

package procmon

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Alive reports whether pid names a live process. Windows has no signal-0
// probe (Process.Signal only supports os.Kill), so existence is checked via
// `tasklist`'s CSV output instead.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH", "/FO", "CSV").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "\""+strconv.Itoa(pid)+"\"")
}

// GroupAlive reports whether the group leader is still alive. Windows has no
// POSIX process group; PGID is always the leader's own PID (Setpgid is not
// set on this platform — see execution_windows.go), so this is Alive(pgid).
func GroupAlive(pgid int) bool {
	return Alive(pgid)
}

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
