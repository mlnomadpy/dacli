//go:build !windows

package procmon

import (
	"syscall"
	"time"
)

// Alive reports whether pid names a live process, via a signal-0 probe (send
// no signal, just test existence/permission). EPERM means it exists but is
// owned elsewhere — still alive for our purposes.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// GroupAlive reports whether ANY member of the process group still exists.
// This is the runaway check: the leader can exit while forked children keep
// running, and the group lives as long as one member does.
func GroupAlive(pgid int) bool {
	if pgid <= 0 {
		return false
	}
	err := syscall.Kill(-pgid, 0)
	return err == nil || err == syscall.EPERM
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
