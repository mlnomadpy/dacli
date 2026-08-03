//go:build !windows

package execution

import (
	"syscall"
	"testing"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

// The dacli-177 regression, isolated: `dacli wait` must NOT call a run finished
// when the group leader has exited but forked children (a commit still being
// written, a helper process) are alive under the same group. AliveRecord alone
// misses that case entirely, so runStillLive has to consult GroupAlive too — if
// it ever collapses back to a bare leader probe, the LAND step proceeds while
// children are mid-commit.
func TestRunStillLiveDetectsLiveGroupWithDeadLeader(t *testing.T) {
	pgid := syscall.Getpgrp() // this test process's group: genuinely alive
	if !procmon.GroupAlive(pgid) {
		t.Skipf("process group %d not observable here", pgid)
	}
	rec := procmon.Record{PID: 1 << 30, PGID: pgid} // leader pid cannot exist
	if procmon.AliveRecord(rec) {
		t.Fatal("test premise broken: the fabricated leader pid must be dead")
	}
	if !runStillLive(rec) {
		t.Error("a live process GROUP with a dead leader must still count as live")
	}
}
