//go:build !windows

package ghmirror

import (
	"os"
	"os/exec"
	"syscall"
)

func ghInterruptSignals() []os.Signal { return []os.Signal{os.Interrupt, syscall.SIGTERM} }

func setGHProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
