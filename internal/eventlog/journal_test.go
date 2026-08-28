package eventlog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

type journalFixture struct {
	w           *workspace.Workspace
	open        *store.Task
	journal     *Event
	terminal    *Event
	missing     *Event
	oldProposal *Event
	newProposal *Event
	actionable  *Event
	referenced  *Event
	corrupt     *Event
}

func makeJournalFixture(t *testing.T) journalFixture {
	t.Helper()
	w, open := setup(t)
	done, err := store.CreateTask(w, "a-root", "core", "done target", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, done, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	appendEvent := func(kind model.EventKind, about, actor, body string) *Event {
		t.Helper()
		event, err := Append(w, actor, kind, about, "", body)
		if err != nil {
			t.Fatal(err)
		}
		return event
	}
	f := journalFixture{w: w, open: open}
	f.journal = appendEvent(model.EventCommit, open.ID, "a-worker", "sha abc")
	f.terminal = appendEvent(model.EventBlock, done.ID, "a-worker", "obsolete terminal block")
	f.missing = appendEvent(model.EventClaim, "t-missing", "a-worker", "missing target")
	f.oldProposal = appendEvent(model.EventProposeStatus, open.ID, "a-one", "propose: active")
	time.Sleep(time.Millisecond)
	f.newProposal = appendEvent(model.EventProposeStatus, open.ID, "a-two", "propose: done")
	f.actionable = appendEvent(model.EventComment, open.ID, "a-worker", "still actionable")
	_ = appendEvent(model.EventClaim, open.ID, "a-one", "competing claim")
	_ = appendEvent(model.EventBlock, open.ID, "a-two", "competing block")
	f.referenced = appendEvent(model.EventHelp, open.ID, "a-worker", "question")
	_ = appendEvent(model.EventAnswer, f.referenced.ID, "a-root", "answer reference")
	f.corrupt = appendEvent(model.EventFinding, open.ID, "a-worker", "corrupt me")
	doc, err := mdstore.ReadFile(f.corrupt.Path)
	if err != nil {
		t.Fatal(err)
	}
	doc.Sections[0].Content = "tampered without checksum\n"
	if err := mdstore.WriteFile(f.corrupt.Path, doc); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestJournalPlanClassifiesEveryEvidenceClassWithoutMutation(t *testing.T) {
	f := makeJournalFixture(t)
	before := eventTreeFiles(t, f.w)
	plan, err := PlanJournal(f.w, "core", []string{"complete-journal"}, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	after := eventTreeFiles(t, f.w)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatal("read-only plan changed event history")
	}
	byID := map[string]JournalItem{}
	classes := map[string]bool{}
	for _, item := range plan.Items {
		byID[item.ID] = item
		classes[item.Classification] = true
	}
	for _, class := range []string{"complete-journal", "terminal-target", "missing-target", "superseded-proposal", "pending-actionable", "externally-referenced", "contested", "unknown-unreadable"} {
		if !classes[class] {
			t.Errorf("missing classification %s: %+v", class, plan.Items)
		}
	}
	if byID[f.journal.ID].Action != "archive" || byID[f.terminal.ID].Action != "dismiss" || byID[f.missing.ID].Action != "dismiss" || byID[f.oldProposal.ID].Action != "dismiss" {
		t.Fatalf("safe actions wrong: %+v", byID)
	}
	for _, event := range []*Event{f.newProposal, f.actionable, f.referenced} {
		if byID[event.ID].Action != "preserve" || byID[event.ID].ManualAction == "" {
			t.Fatalf("unsafe event actionable: %+v", byID[event.ID])
		}
	}
	if plan.ArchiveCount != 1 || plan.DismissCount != 3 || plan.ArchiveBytes == 0 || len(plan.ID) != 64 {
		t.Fatalf("impact/identity incomplete: %+v", plan)
	}
}

func TestJournalPlanClassifiesUnreadableEventSubtreeUnknown(t *testing.T) {
	w, _ := setup(t)
	original := walkEventTree
	walkEventTree = func(root string, fn fs.WalkDirFunc) error { return fn(root, nil, os.ErrPermission) }
	t.Cleanup(func() { walkEventTree = original })
	plan, err := PlanJournal(w, "core", nil, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 || plan.Items[0].Classification != "unknown-unreadable" || plan.Items[0].Action != "preserve" || plan.Items[0].ManualAction == "" {
		t.Fatalf("walk fault presented as empty/actionable: %+v", plan.Items)
	}
}

func TestJournalApplyPreservesOriginalsAndArchiveRemainsQueryable(t *testing.T) {
	f := makeJournalFixture(t)
	plan, err := PlanJournal(f.w, "core", nil, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	original := map[string]string{}
	for _, event := range []*Event{f.terminal, f.missing, f.oldProposal, f.journal} {
		original[event.ID] = hashFile(t, event.Path)
	}
	snapshot, err := ApplyJournalPlan(f.w, "a-root", "core", nil, plan.ID, time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.PlanID != plan.ID || len(snapshot.Items) != len(plan.Items) {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	for _, event := range []*Event{f.terminal, f.missing, f.oldProposal} {
		if hashFile(t, event.Path) != original[event.ID] {
			t.Fatalf("original %s was rewritten", event.ID)
		}
	}
	archivePath := filepath.Join(f.w.Root, workspace.Dir, "events-archive", logicalEventPath(f.w, f.journal.Path))
	if hashFile(t, archivePath) != original[f.journal.ID] {
		t.Fatal("archival move lost or changed journal bytes")
	}
	all, err := List(f.w, Query{})
	if err != nil {
		t.Fatal(err)
	}
	foundArchived := false
	for _, event := range all {
		if event.ID == f.journal.ID {
			foundArchived = true
		}
	}
	if !foundArchived {
		t.Fatal("archived durable provenance is no longer queryable")
	}
	pending, err := List(f.w, Query{Pending: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range pending {
		if event.ID == f.terminal.ID || event.ID == f.missing.ID || event.ID == f.oldProposal.ID {
			t.Fatalf("dismissed obsolete event remains actionable: %s", event.ID)
		}
	}
	projection, err := store.LocalDeliveryProjection(f.w, "core", time.Unix(3, 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range projection.Findings {
		if finding.ObjectID == f.terminal.ID || finding.ObjectID == f.missing.ID || finding.ObjectID == f.oldProposal.ID {
			t.Fatalf("reconciliation still calls dismissed event actionable: %+v", finding)
		}
	}
	if _, err := os.Stat(filepath.Join(f.w.Root, workspace.Dir, "event-journal", plan.ID, "snapshot.json")); err != nil {
		t.Fatalf("snapshot missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.w.Root, workspace.Dir, "event-journal", plan.ID, "snapshot.json.sha256")); err != nil {
		t.Fatalf("snapshot checksum missing: %v", err)
	}
}

func TestJournalConfiguredMailboxArchiveKeepsFindingProvenanceQueryable(t *testing.T) {
	w, task := setup(t)
	finding, err := Append(w, "a-reviewer", model.EventFinding, task.ID, "file:service.go", "durable finding")
	if err != nil {
		t.Fatal(err)
	}
	if err := MarkApplied(finding.Path); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanJournal(w, "core", []string{"complete-mailbox"}, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	item := JournalItem{}
	for _, candidate := range plan.Items {
		if candidate.ID == finding.ID {
			item = candidate
		}
	}
	if item.Classification != "complete-mailbox" || item.Action != "archive" {
		t.Fatalf("finding policy classification = %+v", item)
	}
	if _, err := ApplyJournalPlan(w, "a-root", "core", []string{"complete-mailbox"}, plan.ID, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	event, err := Find(w, finding.ID)
	if err != nil {
		t.Fatal(err)
	}
	if event.Body != "durable finding" || event.Origin != "file:service.go" || !strings.Contains(event.Path, "events-archive") {
		t.Fatalf("archived provenance changed: %+v", event)
	}
}

func TestJournalApplyCrashRestartIsIdempotentAtEveryPhase(t *testing.T) {
	for _, phase := range []string{"snapshot", "dismissal", "archive"} {
		t.Run(phase, func(t *testing.T) {
			f := makeJournalFixture(t)
			plan, err := PlanJournal(f.w, "core", nil, time.Unix(1, 0))
			if err != nil {
				t.Fatal(err)
			}
			injected := errors.New("injected " + phase)
			JournalPhaseHook = func(got string) error {
				if got == phase {
					return injected
				}
				return nil
			}
			_, err = ApplyJournalPlan(f.w, "a-root", "core", nil, plan.ID, time.Unix(2, 0))
			JournalPhaseHook = nil
			if !errors.Is(err, injected) {
				t.Fatalf("phase %s error = %v", phase, err)
			}
			snapshotPath := filepath.Join(f.w.Root, workspace.Dir, "event-journal", plan.ID, "snapshot.json")
			beforeRestart, err := os.ReadFile(snapshotPath)
			if err != nil {
				t.Fatal(err)
			}
			// The same reviewed identity resumes after process restart.
			if _, err := ApplyJournalPlan(f.w, "a-root", "core", nil, plan.ID, time.Unix(3, 0)); err != nil {
				t.Fatalf("restart after %s: %v", phase, err)
			}
			afterRestart, err := os.ReadFile(snapshotPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(afterRestart) != string(beforeRestart) {
				t.Fatalf("restart after %s rewrote immutable snapshot", phase)
			}
			all, _ := List(f.w, Query{Kinds: []model.EventKind{model.EventDismissal}})
			if len(all) != 3 {
				t.Fatalf("restart after %s wrote %d dispositions, want 3", phase, len(all))
			}
		})
	}
}

func eventTreeFiles(t *testing.T, w *workspace.Workspace) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(filepath.Join(w.Root, workspace.Dir), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, path)
		}
		return nil
	})
	return out
}

func hashFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
