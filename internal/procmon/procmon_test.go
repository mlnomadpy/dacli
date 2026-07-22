package procmon_test

import (
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

// The load-bearing guarantee: a spawned agent's WHOLE process tree is
// sampleable and killable as a unit. A shell that forks a background child
// stands in for `claude -p ...` forking its helpers — SIGTERM'ing the group
// must reap the child too, or a runaway leaks resources after dacli moves on.
func TestSampleAndKillReapWholeTree(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 30 & sleep 30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := cmd.Process.Pid // Setpgid ⇒ leader pid == group id
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }() // continuously reap the leader
	defer syscall.Kill(-pgid, syscall.SIGKILL)  // safety net if asserts fail

	// The group should hold the leader plus its forked child.
	var u procmon.Usage
	for i := 0; i < 40; i++ {
		u = procmon.SampleGroup(pgid)
		if u.Procs >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if u.Procs < 2 {
		t.Fatalf("group should hold the agent + its forked child; saw %d proc", u.Procs)
	}
	if u.RSSKB <= 0 {
		t.Errorf("expected nonzero resident memory, got %d KB", u.RSSKB)
	}
	if u.GPUMiB != -1 {
		t.Logf("nvidia GPU present: group holds %d MiB", u.GPUMiB)
	}
	if !procmon.Alive(pgid) || !procmon.GroupAlive(pgid) {
		t.Fatal("group should be alive before the kill")
	}

	termed, _ := procmon.KillTree(pgid, 3*time.Second)
	if !termed {
		t.Fatal("SIGTERM to the group should have landed")
	}
	<-done // leader reaped

	for i := 0; i < 40 && procmon.GroupAlive(pgid); i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if procmon.GroupAlive(pgid) {
		t.Fatal("KillTree left group members alive — runaway children not reaped")
	}
}

func TestRecordRoundTripAndLiveness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proc.txt")
	rec := procmon.Record{
		RunID: "01ABCDEF", Child: "a-1", Task: "t-1", Role: "junior",
		Runtime: "cc", PID: 4242, PGID: 4242, Started: time.Now(),
	}
	if err := procmon.WriteRecord(path, rec); err != nil {
		t.Fatal(err)
	}
	got, err := procmon.ReadRecord(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != rec.RunID || got.Child != rec.Child || got.Runtime != rec.Runtime ||
		got.PID != rec.PID || got.PGID != rec.PGID {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	// Liveness is probed, never assumed: a pid that cannot exist is not alive,
	// and non-positive pids are rejected outright.
	if procmon.Alive(1 << 30) {
		t.Error("implausible pid reported alive")
	}
	if procmon.Alive(0) || procmon.Alive(-1) {
		t.Error("non-positive pid reported alive")
	}
}

// PID-reuse safety: a live PID is only "our agent" while its OS start time still
// matches the one recorded at spawn. A stale record whose PID has been recycled
// onto an unrelated process (simulated by a bogus recorded start time) must NOT
// resurface as live — otherwise `agents`/`kill`/`wait` mis-sample or signal the
// wrong process group.
func TestAliveIdentityRejectsRecycledPID(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	start, ok := procmon.ProcStart(pid)
	if !ok || start == "" {
		t.Fatalf("ProcStart(%d) failed to read a start time", pid)
	}
	// Same PID + the start time we recorded ⇒ still our process.
	if !procmon.AliveIdentity(pid, start) {
		t.Error("AliveIdentity should accept a live PID with a matching start time")
	}
	// Same PID + a start time that does not match ⇒ the number was recycled.
	if procmon.AliveIdentity(pid, "Thu Jan  1 00:00:00 1970") {
		t.Error("AliveIdentity must reject a live PID whose start time differs (recycled)")
	}
	// Empty recorded start time is a legacy record: fall back to a bare probe.
	if !procmon.AliveIdentity(pid, "") {
		t.Error("AliveIdentity with no recorded start should fall back to Alive()")
	}

	// PIDStart survives a proc.txt round-trip (it contains colons, so parsing
	// must keep everything after the first key colon).
	path := filepath.Join(t.TempDir(), "proc.txt")
	if err := procmon.WriteRecord(path, procmon.Record{PID: pid, PGID: pid, PIDStart: start}); err != nil {
		t.Fatal(err)
	}
	got, err := procmon.ReadRecord(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.PIDStart != start {
		t.Errorf("PIDStart round-trip: got %q, want %q", got.PIDStart, start)
	}
	if !procmon.AliveRecord(got) {
		t.Error("AliveRecord should accept the round-tripped live record")
	}
}
