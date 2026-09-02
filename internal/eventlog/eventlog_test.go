package eventlog

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/eventdisp"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestDismissPreservesOriginalAndRemovesItFromPending(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "dismiss")
	if err != nil {
		t.Fatal(err)
	}
	original, err := Append(w, "a-author", model.EventBlock, "t-task", "", "obsolete diagnostic")
	if err != nil {
		t.Fatal(err)
	}
	disposition, created, err := Dismiss(w, "a-root", original, "superseded by recovery support")
	if err != nil || !created {
		t.Fatalf("Dismiss: created=%v err=%v", created, err)
	}
	if disposition.Kind != model.EventDismissal || disposition.About != original.ID || disposition.Actor != "a-root" {
		t.Fatalf("audit disposition lost provenance: %+v", disposition)
	}
	if _, err := os.Stat(original.Path); err != nil {
		t.Fatalf("dismissal deleted the original event: %v", err)
	}
	pending, err := List(w, Query{Pending: true})
	if err != nil || len(pending) != 0 {
		t.Fatalf("dismissed event remains pending: %v err=%v", pending, err)
	}
	all, err := List(w, Query{})
	if err != nil || len(all) != 2 {
		t.Fatalf("append-only history = %d events, want original + disposition (err %v)", len(all), err)
	}
	var gotOriginal *Event
	for _, event := range all {
		if event.ID == original.ID {
			gotOriginal = event
		}
	}
	if gotOriginal == nil || !gotOriginal.Dismissed || gotOriginal.Pending {
		t.Fatalf("original disposition state = %+v", gotOriginal)
	}
	again, created, err := Dismiss(w, "a-root", gotOriginal, "repeat")
	if err != nil || created || again.ID != disposition.ID {
		t.Fatalf("repeated dismissal was not idempotent: event=%+v created=%v err=%v", again, created, err)
	}
}

func TestListBuildsDismissalIndexWithoutReadingTheLogTwice(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "single-pass")
	if err != nil {
		t.Fatal(err)
	}
	original, err := Append(w, "a-worker", model.EventFinding, "t-task", "", "obsolete")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Dismiss(w, "a-root", original, "superseded"); err != nil {
		t.Fatal(err)
	}
	if _, err := Append(w, "a-worker", model.EventComment, "t-task", "", "still pending"); err != nil {
		t.Fatal(err)
	}

	originalReader := readEventFile
	reads := 0
	readEventFile = func(path string) (*mdstore.Doc, error) {
		reads++
		return originalReader(path)
	}
	t.Cleanup(func() { readEventFile = originalReader })

	all, err := List(w, Query{})
	if err != nil {
		t.Fatal(err)
	}
	if reads != len(all) {
		t.Fatalf("event files read %d times for %d events; want one integrity parse per file", reads, len(all))
	}
	for _, event := range all {
		if event.ID == original.ID && !event.Dismissed {
			t.Fatal("single-pass index lost the valid dismissal")
		}
	}
}

func TestDismissRefusesAppliedEvent(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "dismiss-applied")
	if err != nil {
		t.Fatal(err)
	}
	event, err := Append(w, "a-author", model.EventBlock, "t-task", "", "already handled")
	if err != nil {
		t.Fatal(err)
	}
	if err := MarkApplied(event.Path); err != nil {
		t.Fatal(err)
	}
	event, err = Find(w, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Dismiss(w, "a-root", event, "wrong"); err == nil || !strings.Contains(err.Error(), "append a compensating event") {
		t.Fatalf("applied dismissal error = %v", err)
	}
}

func TestCorruptDismissalFailsClosed(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "dismiss-corrupt")
	if err != nil {
		t.Fatal(err)
	}
	original, err := Append(w, "a-author", model.EventBlock, "t-task", "", "still actionable")
	if err != nil {
		t.Fatal(err)
	}
	disposition, _, err := Dismiss(w, "a-root", original, "valid reason")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := mdstore.ReadFile(disposition.Path)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the original checksum while changing its covered payload. A corrupt
	// terminal record must not make a valid proposal disappear.
	doc.Sections[0].Content = "tampered reason\n"
	if err := mdstore.WriteFile(disposition.Path, doc); err != nil {
		t.Fatal(err)
	}

	pending, holes, err := ListReport(w, Query{Pending: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != original.ID {
		t.Fatalf("corrupt dismissal hid original proposal: pending=%+v", pending)
	}
	if len(holes) != 1 || holes[0] != disposition.Path {
		t.Fatalf("corrupt dismissal was not surfaced as an integrity hole: %v", holes)
	}
}

func TestIntegrityValidDismissalWithoutIdentityFailsClosed(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "dismiss-no-id")
	if err != nil {
		t.Fatal(err)
	}
	original, err := Append(w, "a-author", model.EventBlock, "t-task", "", "still actionable")
	if err != nil {
		t.Fatal(err)
	}
	disposition, _, err := Dismiss(w, "a-root", original, "valid reason")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := mdstore.ReadFile(disposition.Path)
	if err != nil {
		t.Fatal(err)
	}
	created, _ := doc.Front.Get("created")
	doc.Front.Set("id", "")
	doc.Front.Set("checksum", eventdisp.Checksum(eventdisp.Payload{
		SchemaVersion: EventSchemaVersion, DocumentKind: model.KindEvent,
		Kind: model.EventDismissal, Created: created, Actor: "a-root",
		About: original.ID, Origin: "agent", Body: "valid reason",
	}))
	if err := mdstore.WriteFile(disposition.Path, doc); err != nil {
		t.Fatal(err)
	}
	pending, holes, err := ListReport(w, Query{Pending: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(holes) != 0 || len(pending) != 1 || pending[0].ID != original.ID {
		t.Fatalf("identity-free dismissal changed pending truth: pending=%+v holes=%v", pending, holes)
	}
}

func TestAppendPersistsVersionAndChecksum(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	ev, err := Append(w, "a-writer", model.EventComment, "task-1", "agent", "durable payload")
	if err != nil {
		t.Fatal(err)
	}
	d, err := mdstore.ReadFile(ev.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := d.Front.Get("schema_version"); got != strconv.Itoa(EventSchemaVersion) {
		t.Fatalf("schema_version = %q, want %d", got, EventSchemaVersion)
	}
	if got, _ := d.Front.Get("checksum"); got == "" {
		t.Fatal("new event has no checksum")
	}

	events, holes, err := ListReport(w, Query{})
	if err != nil || len(holes) != 0 || len(events) != 1 {
		t.Fatalf("read checksummed event: events=%d holes=%v err=%v", len(events), holes, err)
	}
	if events[0].SchemaVersion != EventSchemaVersion || events[0].Checksum == "" {
		t.Fatalf("version/checksum not exposed on parsed event: %+v", events[0])
	}
}

func TestListReadsLegacyUnversionedEvent(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	ev, err := Append(w, "a-old", model.EventFinding, "task-1", "agent", "old payload")
	if err != nil {
		t.Fatal(err)
	}
	d, err := mdstore.ReadFile(ev.Path)
	if err != nil {
		t.Fatal(err)
	}
	d.Front.Delete("schema_version")
	d.Front.Delete("checksum")
	if err := mdstore.WriteFile(ev.Path, d); err != nil {
		t.Fatal(err)
	}

	events, holes, err := ListReport(w, Query{})
	if err != nil || len(holes) != 0 || len(events) != 1 {
		t.Fatalf("legacy read: events=%d holes=%v err=%v", len(events), holes, err)
	}
	if events[0].Body != "old payload" || events[0].SchemaVersion != 0 {
		t.Fatalf("legacy event changed during migration read: %+v", events[0])
	}
}

func TestListRejectsChecksumMismatch(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	ev, err := Append(w, "a-writer", model.EventComment, "task-1", "agent", "original payload")
	if err != nil {
		t.Fatal(err)
	}
	d, err := mdstore.ReadFile(ev.Path)
	if err != nil {
		t.Fatal(err)
	}
	d.Sections[0].Content = "tampered payload\n"
	if err := mdstore.WriteFile(ev.Path, d); err != nil {
		t.Fatal(err)
	}

	events, holes, err := ListReport(w, Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || len(holes) != 1 || holes[0] != ev.Path {
		t.Fatalf("checksum mismatch was not isolated as a corrupt log hole: events=%d holes=%v", len(events), holes)
	}
}

// TestListSurfacesMalformedEvent proves a corrupt event file is not silently
// dropped: the readable event still lists, and the parse failure is logged
// rather than hidden — the append-only log must never erase an event without
// signal.
func TestListSurfacesMalformedEvent(t *testing.T) {
	root := t.TempDir()
	w, err := workspace.Init(root, "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	// One well-formed event via the real append path.
	if _, err := Append(w, "a-x", model.EventComment, "", "", "hello"); err != nil {
		t.Fatalf("append: %v", err)
	}

	// One malformed event file dropped straight into the events tree — an
	// unterminated frontmatter block mdstore.Parse rejects.
	bad := filepath.Join(w.EventsDir(), "2026", "01", "01", "01ZZZ-a-y-comment.md")
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(bad, []byte("---\nid: broken\nno-closing-fence\n"), 0o644); err != nil {
		t.Fatalf("write bad: %v", err)
	}

	var logbuf bytes.Buffer
	log.SetOutput(&logbuf)
	defer log.SetOutput(os.Stderr)

	events, err := List(w, Query{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// The good event still comes through.
	if len(events) != 1 || events[0].Body != "hello" {
		t.Fatalf("expected the readable event to survive, got %+v", events)
	}
	// The bad event is surfaced, not hidden.
	if !strings.Contains(logbuf.String(), bad) {
		t.Fatalf("malformed event was dropped silently; log = %q", logbuf.String())
	}
}
