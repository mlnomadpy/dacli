//go:build windows

package execution

import (
	"os/exec"
	"strconv"
)

// setNewProcessGroup is a no-op on Windows: there is no POSIX process group
// to opt into (Setpgid doesn't exist here). killProcessGroup instead relies
// on taskkill's own parent/child tree tracking.
func setNewProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup force-kills pid and its whole descendant tree via
// taskkill, the Windows analogue of signalling a negative pgid.
func killProcessGroup(pid int) error {
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
}
