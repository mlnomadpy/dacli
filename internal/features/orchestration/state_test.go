package orchestration

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
)

// TestLoopPersistsStateForStatusToRead is the 093 regression: a completed
// cycle must leave behind a snapshot that `dacli loop status` can read back —
// cycle count, trunk marker, tokens spent this window, and the ready backlog
// size — without needing the loop process itself still running.
func TestLoopPersistsStateForStatusToRead(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}

	fr := &fakeRunner{}
	gov := &Governor{MaxCycles: 1, NoProgressHalt: 3}
	d := newDriver(w, fr, gov)
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	st, err := readLoopState(w, "p")
	if err != nil {
		t.Fatalf("expected a persisted loop state, got error: %v", err)
	}
	if st.Cycle != 1 {
		t.Fatalf("want cycle 1 persisted, got %d", st.Cycle)
	}
	if st.Status == "" {
		t.Fatal("want a persisted status decision, got empty")
	}
	if st.UpdatedAt.IsZero() {
		t.Fatal("want a persisted timestamp")
	}

	// `dacli loop status` reads the same snapshot back through the command
	// surface, not just the internal helper.
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdLoopStatus(ctx, nil); err != nil {
		t.Fatalf("loop status: %v", err)
	}
	got := out.String()
	for _, want := range []string{"cycle 1", "trunk marker", "tokens this window", "ready backlog"} {
		if !strings.Contains(got, want) {
			t.Fatalf("loop status output missing %q, got: %s", want, got)
		}
	}
}

// TestLoopStatusErrorsWithoutAPriorLoopRun covers the honest-degrade path: no
// persisted state means the command must say so, not print zeroes as if a
// loop had actually run.
func TestLoopStatusErrorsWithoutAPriorLoopRun(t *testing.T) {
	w := loopEnv(t)
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdLoopStatus(ctx, nil); err == nil {
		t.Fatal("want an error when no loop has ever run for the project")
	}
}

// TestGovernorStateRoundTrips is the 096 acceptance test: cycle/window/streak
// counters written to disk must come back byte-for-byte equivalent, since
// that round trip is what lets a restarted loop resume instead of resetting.
func TestGovernorStateRoundTrips(t *testing.T) {
	w := loopEnv(t)

	g := &Governor{WindowDur: time.Hour, WindowTokens: 1000, NoProgressHalt: 3}
	g.Before(1, time.Unix(1_000_000, 0)) // starts the window
	g.AfterCycle(0, 300)                 // cycle 1, zero progress: streak=1
	g.AfterCycle(0, 300)                 // cycle 2, zero progress: streak=2

	writeGovernorState(w, "p", g.State())

	st, err := readGovernorState(w, "p")
	if err != nil {
		t.Fatalf("expected a persisted governor state, got error: %v", err)
	}

	restored := &Governor{WindowDur: time.Hour, WindowTokens: 1000, NoProgressHalt: 3}
	restored.Restore(st)

	if restored.Cycle() != g.Cycle() {
		t.Fatalf("cycle: want %d, got %d", g.Cycle(), restored.Cycle())
	}
	if restored.WindowSpent() != g.WindowSpent() {
		t.Fatalf("window spent: want %d, got %d", g.WindowSpent(), restored.WindowSpent())
	}
	if !restored.WindowStart().Equal(g.WindowStart()) {
		t.Fatalf("window start: want %s, got %s", g.WindowStart(), restored.WindowStart())
	}
	if restored.ZeroStreak() != g.ZeroStreak() {
		t.Fatalf("zero streak: want %d, got %d", g.ZeroStreak(), restored.ZeroStreak())
	}

	// The restored streak must still be live: one more zero-progress cycle
	// should trip the thrash guard exactly as it would have without a restart.
	if dec, _ := restored.AfterCycle(0, 0); dec != Halt {
		t.Fatalf("want restored streak to carry forward into a halt, got %s", dec)
	}
}

// TestGovernorStateAbsentIsHonestError mirrors readLoopState's degrade path:
// a project that has never checkpointed must error, not fabricate a
// zero-valued governor state.
func TestGovernorStateAbsentIsHonestError(t *testing.T) {
	w := loopEnv(t)
	if _, err := readGovernorState(w, "p"); err == nil {
		t.Fatal("want an error when no governor state has ever been persisted")
	}
}

// TestLoopRestartResumesGovernorState is the end-to-end 096 regression: a
// second `dacli loop` driver constructed the way cmdLoop constructs one
// (reloading persisted state before its first Before()) must pick up where a
// prior process left off — cycle count and thrash streak both carry forward —
// rather than starting a perpetual loop over from zero every restart.
func TestLoopRestartResumesGovernorState(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}

	// First process: one cycle, then the non-yolo checkpoint return (which
	// still persists governor state, exactly like a real restart boundary).
	gov1 := &Governor{MaxCycles: 5, NoProgressHalt: 5}
	d1 := newDriver(w, &fakeRunner{}, gov1)
	if err := d1.loop(); err != nil {
		t.Fatal(err)
	}
	if gov1.Cycle() != 1 {
		t.Fatalf("want first process to complete 1 cycle, got %d", gov1.Cycle())
	}

	// Second process: a fresh Governor, reloaded from disk the way cmdLoop
	// does — this must NOT start back at cycle 0.
	gov2 := &Governor{MaxCycles: 5, NoProgressHalt: 5}
	st, err := readGovernorState(w, "p")
	if err != nil {
		t.Fatalf("expected persisted governor state after the first run: %v", err)
	}
	gov2.Restore(st)
	if gov2.Cycle() != gov1.Cycle() {
		t.Fatalf("restart must resume cycle count: want %d, got %d", gov1.Cycle(), gov2.Cycle())
	}

	d2 := newDriver(w, &fakeRunner{}, gov2)
	if err := d2.loop(); err != nil {
		t.Fatal(err)
	}
	if gov2.Cycle() != 2 {
		t.Fatalf("second process should advance to cycle 2 (not reset to 1), got %d", gov2.Cycle())
	}
}
