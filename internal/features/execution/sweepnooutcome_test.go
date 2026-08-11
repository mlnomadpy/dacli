// A detached run whose process is gone but whose outcome was never computed is
// what `sweepFinishedDetached` exists to finalize — nobody ran `dacli wait` on
// it. The sweep recognised such a run by its "running (detached)" placeholder,
// which meant the ONE case it skipped was the run that never wrote an outcome
// file at all.
//
// That is the worst case, not a lesser one. `dacli agents` lists only live
// processes, so the run is invisible there; `dacli metrics` reads an empty
// outcome as "still running" and excludes it from every rate, silently
// shrinking the completion-rate denominator forever. An agent that ran and left
// nothing then looks exactly like one still working — issue #449.
//
// Measured on this repo's own workspace before the fix: three run directories
// with no outcome.md, one of them still counted as in-flight by metrics months
// after its process died.
package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// deadRun writes a run directory whose recorded process is certainly not
// alive. PID 1 with a start-stamp that cannot match makes AliveIdentity false
// without depending on a pid that happens to be free on the test machine.
func deadRun(t *testing.T, w *workspace.Workspace, outcome string) string {
	t.Helper()
	id := ulid.New()
	dir := w.RunDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{
		RunID: id, Child: "a-ghost-" + id[:6], Task: "t-" + id,
		Role: "go-auditor", PID: 1, PGID: 1,
		PIDStart: "definitely-not-the-real-start-stamp",
	}
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	if procmon.AliveRecord(rec) {
		t.Skip("the synthetic dead record reads as alive on this platform")
	}
	if outcome != "" {
		if err := os.WriteFile(filepath.Join(dir, "outcome.md"), []byte(outcome), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

// TestSweepFinalizesARunThatNeverWroteAnOutcome is the regression.
func TestSweepFinalizesARunThatNeverWroteAnOutcome(t *testing.T) {
	w := newExecWS(t)
	id := deadRun(t, w, "") // no outcome.md at all

	finalized, err := sweepFinishedDetached(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(finalized) != 1 {
		t.Fatalf("a dead run with no outcome file must be finalized, got %v", finalized)
	}
	raw, err := os.ReadFile(filepath.Join(w.RunDir(id), "outcome.md"))
	if err != nil {
		t.Fatalf("the sweep must leave an outcome file behind: %v", err)
	}
	// It must state a RESULT. Writing the placeholder here would convert
	// "never finalized" into "still running" and make the run permanently
	// invisible all over again.
	if strings.HasPrefix(strings.TrimSpace(string(raw)), detachedRunningPlaceholder) {
		t.Fatalf("the sweep wrote the running placeholder as a final outcome:\n%s", raw)
	}
	if !strings.Contains(string(raw), "outcome:") {
		t.Fatalf("the finalized outcome must name an outcome, got:\n%s", raw)
	}
}

// TestSweepFinalizesTheRunningPlaceholder is the case that already worked, kept
// so the switch above cannot regress it while fixing the missing-file case.
func TestSweepFinalizesTheRunningPlaceholder(t *testing.T) {
	w := newExecWS(t)
	deadRun(t, w, detachedRunningPlaceholder+"\nchild: a-ghost\n")

	finalized, err := sweepFinishedDetached(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(finalized) != 1 {
		t.Fatalf("a dead run holding the placeholder must be finalized, got %v", finalized)
	}
}

// TestSweepLeavesAFinalizedRunAlone: re-finalizing would overwrite a real
// verdict with a re-derived one on every `dacli agents`, and the sweep reports
// what it finalized, so a no-op run would print a finalization that did not
// happen.
func TestSweepLeavesAFinalizedRunAlone(t *testing.T) {
	w := newExecWS(t)
	const settled = "outcome: done (detached)\nchild: a-ghost\nacceptance: 2/2\n"
	id := deadRun(t, w, settled)

	finalized, err := sweepFinishedDetached(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(finalized) != 0 {
		t.Fatalf("an already-finalized run must not be swept again, got %v", finalized)
	}
	raw, _ := os.ReadFile(filepath.Join(w.RunDir(id), "outcome.md"))
	if string(raw) != settled {
		t.Fatalf("the settled outcome was rewritten:\n%s", raw)
	}
}

// TestSweepLeavesARunWithNoProcessRecordAlone: with no proc.txt there is no
// liveness to test, so finalizing would be a guess. It stays untouched — and
// deliberately so, rather than by falling through the same branch as the case
// above, which is why it has its own test.
func TestSweepLeavesARunWithNoProcessRecordAlone(t *testing.T) {
	w := newExecWS(t)
	dir := w.RunDir(ulid.New())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invocation.txt"), []byte("dacli spawn ..."), 0o644); err != nil {
		t.Fatal(err)
	}

	finalized, err := sweepFinishedDetached(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(finalized) != 0 {
		t.Fatalf("a run with no process record cannot be finalized against anything, got %v", finalized)
	}
	if _, err := os.Stat(filepath.Join(dir, "outcome.md")); err == nil {
		t.Fatal("the sweep invented an outcome for a run it could not test for liveness")
	}
}

// --- the exit EVENT (#449) ---------------------------------------------
//
// Finalization writes an outcome file, but only when something sweeps — and
// the loop never does. So the record of a run's ending depended on who
// happened to look, and a run nobody looked at stayed indistinguishable from
// one still working. The ending is now a fact in the append-only log.

func exitEvents(t *testing.T, w *workspace.Workspace) []*eventlog.Event {
	t.Helper()
	evs, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventExit}})
	if err != nil {
		t.Fatal(err)
	}
	return evs
}

// TestFinalizeEmitsAnExitEvent is the regression: "found nothing" must be a
// recorded statement, not an absence.
func TestFinalizeEmitsAnExitEvent(t *testing.T) {
	w := newExecWS(t)
	deadRun(t, w, "")

	if _, err := sweepFinishedDetached(w); err != nil {
		t.Fatal(err)
	}
	evs := exitEvents(t, w)
	if len(evs) != 1 {
		t.Fatalf("finalizing a run must record its ending, got %d exit event(s)", len(evs))
	}
	// A run that produced nothing must SAY it produced nothing — silence and
	// success looking identical is the whole complaint.
	if !strings.Contains(evs[0].Body, "no visible result") {
		t.Fatalf("an agent that produced nothing must say so, got: %s", evs[0].Body)
	}
	if !strings.Contains(evs[0].Body, "ended") {
		t.Fatalf("the event must state that the run ended, got: %s", evs[0].Body)
	}
}

// TestExitEventIsWrittenOncePerRun: `dacli agents` sweeps on every invocation,
// so an event written per OBSERVATION rather than per finalization would fill
// the log with duplicates of the same ending.
func TestExitEventIsWrittenOncePerRun(t *testing.T) {
	w := newExecWS(t)
	deadRun(t, w, "")

	for i := 0; i < 3; i++ {
		if _, err := sweepFinishedDetached(w); err != nil {
			t.Fatal(err)
		}
	}
	if evs := exitEvents(t, w); len(evs) != 1 {
		t.Fatalf("three sweeps must record one ending, got %d", len(evs))
	}
}

// TestExitEventIsBornApplied: an exit event is a journal record — nothing acts
// on it. Born pending it would sit in the "work waiting for someone" count
// forever, which is exactly what happened to 203 commit events.
func TestExitEventIsBornApplied(t *testing.T) {
	if !model.EventExit.IsJournal() {
		t.Fatal("EventExit must be a journal kind, or every finalized run adds permanent pending work")
	}
	w := newExecWS(t)
	deadRun(t, w, "")
	if _, err := sweepFinishedDetached(w); err != nil {
		t.Fatal(err)
	}
	pending, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventExit}, Pending: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("exit events must not be born pending, got %d", len(pending))
	}
}

// TestExitEventNamesTheTaskAndRole so `dacli events tail` and the contrib
// rollup can attribute an ending without opening the run directory.
func TestExitEventNamesTheTaskAndRole(t *testing.T) {
	w := newExecWS(t)
	deadRun(t, w, "")

	if _, err := sweepFinishedDetached(w); err != nil {
		t.Fatal(err)
	}
	evs := exitEvents(t, w)
	if len(evs) != 1 {
		t.Fatalf("got %d exit event(s)", len(evs))
	}
	if !strings.Contains(evs[0].Body, "go-auditor") {
		t.Fatalf("the ending must name the role, got: %s", evs[0].Body)
	}
	if evs[0].About == "" {
		t.Fatalf("the ending must name the task it was working, got about=%q", evs[0].About)
	}
}
