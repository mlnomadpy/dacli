package procmon_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

// PathsOverlap is what keeps parallel agents merge-clean: two live agents must
// never claim the same tree. The rule is path-SEGMENT containment, so the two
// failure modes it has to avoid are symmetric — declaring a real overlap
// disjoint (parallel agents clobber each other) and declaring a disjoint pair
// overlapping (every spawn deadlocks on a sibling directory).
func TestPathsOverlap(t *testing.T) {
	cases := []struct {
		name string
		a, b []string
		want bool
	}{
		{"identical paths", []string{"internal/store"}, []string{"internal/store"}, true},
		{"a file inside a claimed dir", []string{"internal/store"}, []string{"internal/store/roles.go"}, true},
		{"the containment is symmetric", []string{"internal/store/roles.go"}, []string{"internal/store"}, true},
		{"a deep descendant", []string{"internal"}, []string{"internal/features/execution/execution.go"}, true},
		{"a sibling directory with a shared prefix does NOT overlap",
			[]string{"internal/store"}, []string{"internal/storefront"}, false},
		{"a shared prefix at the top level does NOT overlap",
			[]string{"internal"}, []string{"internals"}, false},
		{"unrelated trees", []string{"internal/store"}, []string{"cmd/dacli"}, false},
		{"any overlapping pair in the lists is enough",
			[]string{"docs", "internal/store"}, []string{"cmd", "internal/store/roles.go"}, true},
		{"no overlapping pair", []string{"docs", "cmd"}, []string{"internal", "scripts"}, false},
		{"trailing slashes are normalized away",
			[]string{"internal/store/"}, []string{"/internal/store"}, true},
		{"surrounding whitespace is normalized away",
			[]string{" internal/store "}, []string{"internal/store/roles.go"}, true},
		{"an empty claim list never conflicts", nil, []string{"internal/store"}, false},
		{"both empty", nil, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mine, theirs, got := procmon.PathsOverlap(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("PathsOverlap(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
			// A conflict refusal quotes both sides; empty strings would make the
			// message useless to the agent that has to narrow its scope.
			if got && (mine == "" || theirs == "") {
				t.Errorf("a reported conflict must name both paths, got (%q, %q)", mine, theirs)
			}
		})
	}
}

// A proc.txt is the only handle a SEPARATE dacli invocation has on a live tree,
// so every field has to survive the round-trip — including the multi-value
// claims list and the colon-bearing start time.
func TestRecordRoundTripKeepsClaimsAndStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proc.txt")
	want := procmon.Record{
		RunID: "01RUNZZZ", Child: "a-42", Task: "t-7", Role: "implementer", Runtime: "claude-code",
		PID: 1234, PGID: 1234, PIDStart: "Mon Aug  3 09:15:00 2026",
		Started: time.Now().UTC().Truncate(time.Second),
		Claims:  []string{"internal/store", "internal/features/execution"},
	}
	if err := procmon.WriteRecord(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := procmon.ReadRecord(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != want.RunID || got.Child != want.Child || got.Task != want.Task ||
		got.Role != want.Role || got.Runtime != want.Runtime ||
		got.PID != want.PID || got.PGID != want.PGID || got.PIDStart != want.PIDStart {
		t.Errorf("scalar round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
	if !got.Started.Equal(want.Started) {
		t.Errorf("started = %v, want %v", got.Started, want.Started)
	}
	if len(got.Claims) != 2 || got.Claims[0] != "internal/store" || got.Claims[1] != "internal/features/execution" {
		t.Errorf("claims round-trip = %v", got.Claims)
	}
	// A record with no claims must read back as no claims, not as one empty
	// claim — an empty claim is a prefix of every path and would block all spawns.
	bare := filepath.Join(t.TempDir(), "bare.txt")
	if err := procmon.WriteRecord(bare, procmon.Record{RunID: "r", PID: 1, PGID: 1}); err != nil {
		t.Fatal(err)
	}
	got2, err := procmon.ReadRecord(bare)
	if err != nil {
		t.Fatal(err)
	}
	if len(got2.Claims) != 0 {
		t.Errorf("a claimless record read back %d claim(s): %v", len(got2.Claims), got2.Claims)
	}
	if _, _, clash := procmon.PathsOverlap(got2.Claims, []string{"anything"}); clash {
		t.Error("a claimless record must not conflict with every path")
	}
}

// A missing proc.txt is an error the caller skips, never a zero Record treated
// as a live agent.
func TestReadRecordMissingFileIsAnError(t *testing.T) {
	if _, err := procmon.ReadRecord(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Fatal("reading a missing proc.txt must error")
	}
	// Junk lines are skipped rather than aborting the parse: a partially
	// written record still yields the pid needed to reap the tree.
	path := filepath.Join(t.TempDir(), "proc.txt")
	if err := os.WriteFile(path, []byte("garbage line with no colon\npid: 99\nunknown_key: v\npgid: 99\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, err := procmon.ReadRecord(path)
	if err != nil {
		t.Fatal(err)
	}
	if rec.PID != 99 || rec.PGID != 99 {
		t.Errorf("a partially-parseable record lost its pid: %+v", rec)
	}
}

// Every liveness and signalling entry point must reject non-positive ids
// outright. Signalling pgid 0 would target the CALLER's own process group —
// dacli killing itself and its shell.
func TestNonPositiveIDsAreNeverAlive(t *testing.T) {
	for _, id := range []int{0, -1, -1234} {
		if procmon.Alive(id) {
			t.Errorf("Alive(%d) = true", id)
		}
		if procmon.GroupAlive(id) {
			t.Errorf("GroupAlive(%d) = true", id)
		}
		if termed, killed := procmon.KillTree(id, time.Second); termed || killed {
			t.Errorf("KillTree(%d) signalled something (termed=%v killed=%v) — that would target the caller's own group",
				id, termed, killed)
		}
		if _, ok := procmon.ProcStart(id); ok {
			t.Errorf("ProcStart(%d) claimed a start time", id)
		}
		if procmon.AliveIdentity(id, "whatever") {
			t.Errorf("AliveIdentity(%d) = true", id)
		}
	}
}

// SampleGroup must report unmeasurable GPU as -1 (rendered "n/a"), never a
// fabricated 0, and must return an empty snapshot for an invalid group rather
// than parsing the whole process table.
func TestSampleGroupInvalidGroup(t *testing.T) {
	for _, pgid := range []int{0, -1} {
		u := procmon.SampleGroup(pgid)
		if u.Procs != 0 || u.RSSKB != 0 || u.CPUPct != 0 {
			t.Errorf("SampleGroup(%d) = %+v, want an empty snapshot", pgid, u)
		}
		if u.GPUMiB != -1 {
			t.Errorf("SampleGroup(%d).GPUMiB = %d, want -1 (unmeasurable, never a fabricated 0)", pgid, u.GPUMiB)
		}
	}
	// A plausible but nonexistent group yields nothing, not a panic.
	if u := procmon.SampleGroup(1 << 30); u.Procs != 0 {
		t.Errorf("SampleGroup on a nonexistent group found %d proc", u.Procs)
	}
}

// A record whose PID is live but whose recorded start time cannot be confirmed
// is NOT vouched for. The conservative direction matters: AliveRecord feeds
// KillTree, so a wrong "yes" signals a stranger's process group.
func TestAliveRecordUsesIdentityNotBarePID(t *testing.T) {
	pid := os.Getpid()
	start, ok := procmon.ProcStart(pid)
	if !ok {
		t.Skip("ps cannot read this process's start time")
	}
	if !procmon.AliveRecord(procmon.Record{PID: pid, PIDStart: start}) {
		t.Error("a live record with a matching start time must be alive")
	}
	if procmon.AliveRecord(procmon.Record{PID: pid, PIDStart: "Thu Jan  1 00:00:00 1970"}) {
		t.Error("a recycled PID (start time mismatch) must not be reported as our agent")
	}
	if procmon.AliveRecord(procmon.Record{PID: 1 << 30, PIDStart: start}) {
		t.Error("a dead pid must not be alive whatever the recorded start time")
	}
}
