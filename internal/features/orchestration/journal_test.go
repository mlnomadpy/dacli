package orchestration

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
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

func mustWriteCycleJournal(t *testing.T, w *workspace.Workspace, project string, journal cycleJournal) {
	t.Helper()
	if err := writeCycleJournal(w, project, journal); err != nil {
		t.Fatalf("write cycle journal: %v", err)
	}
}

func TestCycleJournalPersistenceFailuresAreReturned(t *testing.T) {
	w := journalWS(t)
	want := cycleJournal{
		PendingAccept:   []pendingAccept{{Seq: 4, Branch: "dacli/004-fix"}},
		PendingLand:     []string{"dacli/record"},
		WindowTokens:    9000,
		Landing:         model.LandingPolicy{Mode: model.LandingPR, Base: "main"},
		LandingExplicit: true,
	}
	sentinel := errors.New("injected persistence failure")

	t.Run("directory create", func(t *testing.T) {
		old := cycleJournalMkdirAll
		cycleJournalMkdirAll = func(string, os.FileMode) error { return sentinel }
		t.Cleanup(func() { cycleJournalMkdirAll = old })
		if err := writeCycleJournal(w, "create", want); !errors.Is(err, sentinel) {
			t.Fatalf("writeCycleJournal error = %v, want injected create error", err)
		}
	})

	for _, stage := range []string{"write", "rename"} {
		t.Run(stage, func(t *testing.T) {
			old := cycleJournalWrite
			cycleJournalWrite = func(string, string) error { return sentinel }
			t.Cleanup(func() { cycleJournalWrite = old })
			if err := writeCycleJournal(w, stage, want); !errors.Is(err, sentinel) {
				t.Fatalf("writeCycleJournal error = %v, want injected %s error", err, stage)
			}
		})
	}

	t.Run("empty ledger removal", func(t *testing.T) {
		if err := writeCycleJournal(w, "remove", want); err != nil {
			t.Fatal(err)
		}
		old := cycleJournalRemove
		cycleJournalRemove = func(string) error { return sentinel }
		t.Cleanup(func() { cycleJournalRemove = old })
		if err := writeCycleJournal(w, "remove", cycleJournal{}); !errors.Is(err, sentinel) {
			t.Fatalf("writeCycleJournal error = %v, want injected removal error", err)
		}
	})
}

// A backend can lose an atomic replacement yet report success (for example a
// faulty adapter). The checkpoint must validate the committed record: otherwise
// the next process sees no pending task, no held record push, and no token cap.
func TestFailedCheckpointCannotSilentlyLoseRestartLedger(t *testing.T) {
	w := journalWS(t)
	want := cycleJournal{
		PendingAccept:   []pendingAccept{{Seq: 8, Branch: "dacli/008-work"}},
		PendingLand:     []string{"dacli/record"},
		WindowTokens:    12000,
		Landing:         model.LandingPolicy{Mode: model.LandingPR, Base: "main"},
		LandingExplicit: true,
	}
	old := cycleJournalWrite
	cycleJournalWrite = func(string, string) error { return nil }
	t.Cleanup(func() { cycleJournalWrite = old })

	if err := writeCycleJournal(w, "restart", want); err == nil {
		t.Fatal("checkpoint succeeded although recovery ledger was not replaced")
	}
	got, warnings := readCycleJournal(w, "restart")
	if !got.empty() || len(warnings) != 0 {
		t.Fatalf("failed checkpoint unexpectedly persisted %+v (warnings %v)", got, warnings)
	}
}

func TestSaveStatePropagatesJournalFailureButSnapshotsRemainAdvisory(t *testing.T) {
	w := journalWS(t)
	d := newDriver(w, &fakeRunner{}, &Governor{WindowTokens: 5000})
	d.pendingAccept = []pendingAccept{{Seq: 2, Branch: "dacli/002-work"}}
	d.pendingLand = []string{"dacli/record"}
	d.cfg.landing = model.LandingPolicy{Mode: model.LandingLocal, Base: "main"}
	d.cfg.landingExplicit = true

	sentinel := errors.New("injected persistence failure")
	oldJournal := cycleJournalWrite
	cycleJournalWrite = func(string, string) error { return sentinel }
	if err := d.saveState("continue", "checkpoint", 1); !errors.Is(err, sentinel) {
		t.Fatalf("saveState error = %v, want journal error", err)
	}
	cycleJournalWrite = oldJournal

	oldLoop, oldGovernor := writeLoopStateFile, writeGovernorStateFile
	writeLoopStateFile = func(string, string) error { return sentinel }
	writeGovernorStateFile = func(string, string) error { return sentinel }
	t.Cleanup(func() {
		cycleJournalWrite = oldJournal
		writeLoopStateFile, writeGovernorStateFile = oldLoop, oldGovernor
	})
	if err := d.saveState("continue", "checkpoint", 1); err != nil {
		t.Fatalf("advisory snapshot failure stopped durable checkpoint: %v", err)
	}
	got, warnings := readCycleJournal(w, "p")
	want := cycleJournal{PendingAccept: d.pendingAccept, PendingLand: d.pendingLand, WindowTokens: 5000, Landing: d.cfg.landing, LandingExplicit: true}
	if len(warnings) != 0 || !reflect.DeepEqual(got, want) {
		t.Fatalf("durable journal = %+v warnings=%v, want %+v", got, warnings, want)
	}
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
	mustWriteCycleJournal(t, w, "core", want)

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

func TestCycleJournalRoundTripsGenerationAndReadsLegacyPendingAccept(t *testing.T) {
	w := journalWS(t)
	mustWriteCycleJournal(t, w, "core", cycleJournal{PendingAccept: []pendingAccept{{
		Seq: 4, Branch: "dacli/004-fix", Generation: 2, GenerationSet: true, VerifyRequired: true,
	}}})
	got, warnings := readCycleJournal(w, "core")
	if len(warnings) != 0 || len(got.PendingAccept) != 1 {
		t.Fatalf("generation journal failed: got=%+v warnings=%v", got, warnings)
	}
	p := got.PendingAccept[0]
	if p.Generation != 2 || !p.GenerationSet || !p.VerifyRequired {
		t.Fatalf("generation or recovery action did not round-trip: %+v", p)
	}

	path := journalFile(w, "core")
	if err := os.WriteFile(path, []byte("pending_accept: 4 dacli/004-fix verify-required\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy, warnings := readCycleJournal(w, "core")
	if len(warnings) != 0 || len(legacy.PendingAccept) != 1 || legacy.PendingAccept[0].GenerationSet || !legacy.PendingAccept[0].VerifyRequired {
		t.Fatalf("legacy pending_accept is not backward compatible: got=%+v warnings=%v", legacy, warnings)
	}
}

// An empty ledger REMOVES the file. A stale one would hold the record push
// hostage forever after the work it described had landed.
func TestCycleJournalEmptyRemovesTheFile(t *testing.T) {
	w := journalWS(t)
	mustWriteCycleJournal(t, w, "core", cycleJournal{PendingLand: []string{"b"}})
	if _, err := os.Stat(journalFile(w, "core")); err != nil {
		t.Fatalf("journal should exist after a non-empty write: %v", err)
	}
	mustWriteCycleJournal(t, w, "core", cycleJournal{})
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
	mustWriteCycleJournal(t, w, "core", cycleJournal{PendingAccept: []pendingAccept{{Seq: 3, Branch: "dacli/003-ok"}}})
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
