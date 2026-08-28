//go:build windows

package ghmirror

import (
	"os"
	"os/exec"
)

func ghInterruptSignals() []os.Signal { return []os.Signal{os.Interrupt} }

// procmon.KillTree uses taskkill /T on Windows, where there is no POSIX
// process-group attribute to set on the child.
func setGHProcessGroup(cmd *exec.Cmd) {}
