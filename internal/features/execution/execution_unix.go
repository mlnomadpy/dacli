//go:build !windows

package execution

import (
	"os/exec"
	"syscall"
)

// setNewProcessGroup makes cmd's child the leader of a new process group
// (Setpgid), so every subprocess it forks inherits a killable group id.
func setNewProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup SIGKILLs the whole group led by pid — the negative pid
// reaches every member, not just the leader.
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
