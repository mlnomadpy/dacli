//go:build !windows

package execution

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

// TestRunStillLivePreservesTask177AfterLeaderExit is the Unix regression for
// task 177, retained alongside task 369's recycled-group guard. The recorded
// leader really forks a helper in its process group and exits; reconciliation
// must keep reporting that authenticated descendant until the helper exits.
func TestRunStillLivePreservesTask177AfterLeaderExit(t *testing.T) {
	helperFile := filepath.Join(t.TempDir(), "helper.pid")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	leader := exec.Command("sh", "-c", `sleep 30 & echo $! > "$1"; read hold || true`, "sh", helperFile)
	leader.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	leader.Stdin = reader
	if err := leader.Start(); err != nil {
		t.Fatal(err)
	}
	// The leader is held alive until its durable identity is recorded. Sandboxed
	// test runners may deny ps; the exact stamp is immaterial after the leader
	// exits, but the record must remain explicitly authenticated rather than
	// exercising the legacy no-identity path.
	pidStart, ok := procmon.ProcStart(leader.Process.Pid)
	if !ok {
		pidStart = "recorded start identity"
	}
	_ = reader.Close()
	_ = writer.Close()
	if err := leader.Wait(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(helperFile)
	if err != nil {
		t.Fatal(err)
	}
	helperPID, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.Kill(helperPID, syscall.SIGKILL) }()

	path := filepath.Join(t.TempDir(), "proc.txt")
	want := procmon.Record{PID: leader.Process.Pid, PGID: leader.Process.Pid, PIDStart: pidStart}
	if err := procmon.WriteRecord(path, want); err != nil {
		t.Fatal(err)
	}
	rec, err := procmon.ReadRecord(path)
	if err != nil {
		t.Fatal(err)
	}
	if procmon.Alive(rec.PID) {
		t.Fatal("test premise broken: recorded group leader is still alive")
	}
	if !procmon.GroupAlive(rec.PGID) {
		t.Fatal("test premise broken: forked helper did not survive in the recorded group")
	}
	if !runStillLive(rec) {
		t.Fatal("task 177: reconciliation lost a genuine helper after its recorded leader exited")
	}
	if err := syscall.Kill(helperPID, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(2 * time.Second); procmon.GroupAlive(rec.PGID) && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if runStillLive(rec) {
		t.Fatal("finished helper left the run reported live")
	}
}

// Task 285 rejected a live PID whose start time no longer matched, but left a
// fallback that trusted GroupAlive after the recorded leader was dead. Model
// that residual with a certainly-dead leader and this test process's unrelated,
// certainly-live group: the numeric group alone cannot authenticate the run.
func TestRunStillLiveRejectsTask369DeadLeaderReusedGroup(t *testing.T) {
	pgid := syscall.Getpgrp() // this test process's unrelated group: genuinely alive
	if !procmon.GroupAlive(pgid) {
		t.Skipf("process group %d not observable here", pgid)
	}
	rec := procmon.Record{PID: 1 << 30, PGID: pgid}
	if procmon.AliveRecord(rec) {
		t.Fatal("test premise broken: the fabricated recorded leader must be dead")
	}
	if runStillLive(rec) {
		t.Error("task 285 residual: an unrelated live group must not resurrect a run whose recorded leader is dead")
	}
}

func TestKillOnePreservesTask369ReusedGroupSafety(t *testing.T) {
	stranger := exec.Command("sh", "-c", "exec sleep 30")
	stranger.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := stranger.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := stranger.Process.Pid
	defer func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		_ = stranger.Wait()
	}()

	w := newExecWS(t)
	ctx, out, _ := newCtx(w.Root)
	rec := procmon.Record{RunID: runID(0), Child: "a-finished", PID: 1 << 30, PGID: pgid}
	killOne(ctx, w, rec, 10*time.Millisecond)

	if !procmon.Alive(pgid) {
		t.Fatal("killOne signalled an unrelated process group that reused a finished run's numeric PGID")
	}
	if !strings.Contains(out.String(), "nothing to signal (already gone)") {
		t.Errorf("killOne did not report the dead recorded run: %q", out.String())
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
