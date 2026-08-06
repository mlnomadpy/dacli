//go:build !windows

package execution

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

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

// task 285: the recycled-leader hole. runStillLive OR's leader-alive with
// GroupAlive so a leader that exited but left live children still counts as
// live (dacli 177). But when the leader PID is ALIVE and RECYCLED — the kernel
// handed the number to an unrelated process whose start time differs from what
// we recorded — rec.PGID names that stranger's group, not our finished agent's.
// The old fallback trusted GroupAlive(rec.PGID) and resurrected the dead run as
// live; it must instead refuse the moment the live leader fails the identity
// check, mirroring procmon.AliveIdentity's leader guard.
func TestRunStillLiveRejectsAliveRecycledLeader(t *testing.T) {
	pid := os.Getpid()        // a genuinely LIVE process...
	pgid := syscall.Getpgrp() // ...whose group is genuinely alive
	if !procmon.GroupAlive(pgid) {
		t.Skipf("process group %d not observable here", pgid)
	}
	// The record names that live PID but a DIFFERENT start time: the identity
	// check fails though Alive(pid) is true — exactly a recycled leader. Its
	// PGID is this live group, so the pre-285 GroupAlive fallback would wrongly
	// call the run live.
	rec := procmon.Record{PID: pid, PGID: pgid, PIDStart: "Wed Jan  1 00:00:00 2020"}
	if procmon.AliveRecord(rec) {
		t.Fatal("test premise broken: a mismatched start time must fail the identity check")
	}
	if !procmon.Alive(rec.PID) {
		t.Fatal("test premise broken: the recorded PID must name a live process")
	}
	if runStillLive(rec) {
		t.Error("a live but recycled leader must make runStillLive false, not be resurrected via GroupAlive")
	}
}

// The prune sibling of the above: `runs prune` must NOT print its "still live …
// pruning it would orphan a running agent" skip for a recycled-leader record —
// that run is dead, and a false skip would pin a ghost's transcript on disk
// forever and mislead an operator into thinking an agent is still running.
func TestRunsPruneDoesNotSkipRecycledLeader(t *testing.T) {
	pid := os.Getpid()
	pgid := syscall.Getpgrp()
	if !procmon.GroupAlive(pgid) {
		t.Skipf("process group %d not observable here", pgid)
	}
	w := newExecWS(t)
	for i := 0; i < 4; i++ {
		mkRun(t, w, runID(i), "outcome: ok\n")
	}
	// runID(0) is the oldest, squarely inside the prune window; give it a
	// recycled-leader record (live PID, mismatched start time, live group).
	dir := w.RunDir(runID(0))
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{
		RunID: runID(0), Child: "a-recycled", PID: pid, PGID: pgid,
		PIDStart: "Wed Jan  1 00:00:00 2020", Started: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdRunsPrune(ctx, []string{"--keep", "1"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "still live") || strings.Contains(out.String(), "a-recycled") {
		t.Errorf("a recycled-leader ghost was reported as a live run to keep:\n%s", out)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Error("the recycled-leader run is dead and must be pruned, not kept")
	}
}
