package eventlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// A corrupt event file is a HOLE in an append-only log, and the reader has to
// be able to tell. List logs it and keeps going — right, since one bad file
// must not blind a reader to the whole log — but the caller got a shorter
// slice and no signal, so a sync over four pending proposals with one corrupt
// file applied three and reported success (dacli 350).
func TestListReportNamesTheEventsItCouldNotRead(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := Append(w, "a-root", model.EventFinding, "", "", "body"); err != nil {
			t.Fatal(err)
		}
	}
	good, holes, err := ListReport(w, Query{})
	if err != nil || len(good) != 3 || len(holes) != 0 {
		t.Fatalf("healthy log: %d events, %d holes, err=%v", len(good), len(holes), err)
	}

	// The shape an interrupted write leaves if the atomic rename is ever
	// bypassed: frontmatter that never terminates.
	bad := filepath.Join(filepath.Dir(good[0].Path), "01CORRUPT0000000000000000-x-finding.md")
	if err := os.WriteFile(bad, []byte("---\nid: 01CORRUPT\nunterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	after, holes, err := ListReport(w, Query{})
	if err != nil {
		t.Fatalf("one bad file must not fail the whole read: %v", err)
	}
	if len(after) != 3 {
		t.Errorf("the readable events must still be returned, got %d", len(after))
	}
	if len(holes) != 1 || !strings.Contains(holes[0], "01CORRUPT") {
		t.Fatalf("the unreadable event must be NAMED to the caller, got %v", holes)
	}
}

// Writes are atomic (mdstore.WriteFile is CreateTemp + Rename), so an
// interrupted write leaves the previous file intact and a stray temp behind —
// never a truncated event. This pins that property, and that the stray temp is
// not mistaken for an event.
func TestAnInterruptedWriteLeavesTheLogReadable(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	first, err := Append(w, "a-root", model.EventFinding, "", "", "the durable one")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(first.Path)

	// The debris a process killed mid-write leaves behind.
	if err := os.WriteFile(filepath.Join(dir, ".dacli-tmp-halfwritten"), []byte("---\nid: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	events, holes, err := ListReport(w, Query{})
	if err != nil {
		t.Fatalf("a stray temp file must not fail the read: %v", err)
	}
	if len(events) != 1 || events[0].ID != first.ID {
		t.Errorf("the durable event must survive intact, got %v", events)
	}
	if len(holes) != 0 {
		t.Errorf("a temp file is not an event and must not be reported as a hole: %v", holes)
	}
}
