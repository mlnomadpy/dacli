//go:build !windows

package execution

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func interruptSignals() []os.Signal { return []os.Signal{os.Interrupt, syscall.SIGTERM} }

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

// procmonGroupHasOtherLiveProcess keeps the guardian present until the runtime
// and its inherited process group drain. Zombies have completed and do not
// extend the run. Failure to inspect is fail-closed: the guardian exits rather
// than manufacturing permanent liveness from an unreadable process table.
func procmonGroupHasOtherLiveProcess(guardianPID int) bool {
	pgid, err := syscall.Getpgid(guardianPID)
	if err != nil {
		return false
	}
	ps := exec.Command("ps", "-A", "-o", "pgid=,pid=,state=")
	// The sampler is itself the guardian's child. Put it in a different group
	// or every snapshot observes the sampler as a live guarded descendant and
	// the guardian can never finish.
	ps.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	out, err := ps.Output()
	if err != nil {
		return false
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		group, _ := strconv.Atoi(fields[0])
		pid, _ := strconv.Atoi(fields[1])
		if group == pgid && pid != guardianPID && !strings.HasPrefix(fields[2], "Z") {
			return true
		}
	}
	return false
}
