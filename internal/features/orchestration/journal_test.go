package orchestration

import (
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func journalWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "j")
	if err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	return w
}

// The ledger must survive the process, because the DEFAULT loop mode returns
// at every checkpoint. Before this, pendingAccept/pendingLand died with the
// process, so the next invocation re-picked tasks whose PRs had merged and
// pushed the record out from under PRs still in flight.
func TestCycleJournalRoundTripsTheLandingLedger(t *testing.T) {
	w := journalWS(t)
	want := cycleJournal{
		PendingAccept: []pendingAccept{{Seq: 12, Branch: "dacli/012-a-slug"}, {Seq: 7, Branch: "dacli/007-x"}},
		PendingLand:   []string{"dacli/record-1"},
		WindowTokens:  500000,
	}
	writeCycleJournal(w, "core", want)

	got, warn := readCycleJournal(w, "core")
	if len(warn) != 0 {
		t.Fatalf("clean journal produced warnings: %v", warn)
	}
	if len(got.PendingAccept) != 2 || got.PendingAccept[0] != want.PendingAccept[0] || got.PendingAccept[1] != want.PendingAccept[1] {
		t.Errorf("pendingAccept = %+v, want %+v", got.PendingAccept, want.PendingAccept)
	}
	if len(got.PendingLand) != 1 || got.PendingLand[0] != "dacli/record-1" {
		t.Errorf("pendingLand = %v, want [dacli/record-1]", got.PendingLand)
	}
	// The CEILING, not the spend: a restart that omits --window-tokens used to
	// restore the spend and then run uncapped.
	if got.WindowTokens != 500000 {
		t.Errorf("window ceiling = %d, want 500000", got.WindowTokens)
	}
}

// An empty ledger REMOVES the file. A stale one would hold the record push
// hostage forever after the work it described had landed.
func TestCycleJournalEmptyRemovesTheFile(t *testing.T) {
	w := journalWS(t)
	writeCycleJournal(w, "core", cycleJournal{PendingLand: []string{"b"}})
	if _, err := os.Stat(journalFile(w, "core")); err != nil {
		t.Fatalf("journal should exist after a non-empty write: %v", err)
	}
	writeCycleJournal(w, "core", cycleJournal{})
	if _, err := os.Stat(journalFile(w, "core")); !os.IsNotExist(err) {
		t.Errorf("an empty ledger must remove the file, got err=%v", err)
	}
	// And a missing file reads clean, not as an error: it is the common case.
	if j, warn := readCycleJournal(w, "core"); !j.empty() || warn != nil {
		t.Errorf("absent journal = %+v warn=%v, want empty and silent", j, warn)
	}
}

// A torn line degrades to "skip that entry, say so" rather than halting the
// loop: unlike the governor snapshot (where resuming with reset guards defeats
// the token ceiling), reconciliation is idempotent, so a partial ledger is
// safe — but a SILENTLY partial one looks exactly like "nothing outstanding".
func TestCycleJournalReportsWhatItDropped(t *testing.T) {
	w := journalWS(t)
	writeCycleJournal(w, "core", cycleJournal{PendingAccept: []pendingAccept{{Seq: 3, Branch: "dacli/003-ok"}}})
	path := journalFile(w, "core")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(raw, []byte("pending_accept: notanumber dacli/x\npending_land:\nbogus\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	got, warn := readCycleJournal(w, "core")
	if len(got.PendingAccept) != 1 || got.PendingAccept[0].Seq != 3 {
		t.Errorf("the good entry must survive: %+v", got.PendingAccept)
	}
	if len(warn) != 3 {
		t.Errorf("warnings = %v, want one per dropped line (bad seq, empty branch, no separator)", warn)
	}
}
