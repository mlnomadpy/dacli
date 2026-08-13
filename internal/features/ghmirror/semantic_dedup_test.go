package ghmirror

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
)

func semanticTestNote(id, title, body string) noteFile {
	d := &mdstore.Doc{Sections: []mdstore.Section{{Level: 1, Title: title}, {Content: body}}}
	d.Front.Set("id", id)
	return noteFile{id: id, title: title, doc: d}
}

// Task 437: repeated recovery reports commonly differ only by incidental title
// wording. They must become one remote record, while the concrete evidence from
// every local record remains in the canonical body.
func TestCanonicalNoteFilesCollapseNearDuplicateTitlesAndKeepEvidence(t *testing.T) {
	notes := []noteFile{
		semanticTestNote("f-one", "Task 435 remote handoff blocked by GitHub DNS", "commit a588ad7: Could not resolve host github.com"),
		semanticTestNote("f-two", "GitHub DNS blocks task 435 remote handoff", "commit 6149670: branch remains local"),
		semanticTestNote("f-three", "MCP schema validation rejects unsupported versions", "schema_version is required"),
	}

	got := canonicalNoteFiles(notes)
	if len(got) != 2 {
		t.Fatalf("canonicalNoteFiles returned %d records, want 2", len(got))
	}
	merged := noteFileText(got[0])
	for _, evidence := range []string{"a588ad7", "6149670"} {
		if !strings.Contains(merged, evidence) {
			t.Fatalf("canonical record lost distinct evidence %q:\n%s", evidence, merged)
		}
	}
}

// Marker-bearing records are the recovery authority. Canonicalization must keep
// the first stable id so a replay adopts the same remote object instead of
// inventing a new semantic marker.
func TestCanonicalNoteFilesKeepStableMarkerIdentity(t *testing.T) {
	w := mirrorWorkspace(t)
	got := canonicalNoteFiles([]noteFile{
		semanticTestNote("d-recovery-one", "Recover partial GitHub push by marker", "first attempt stopped after create"),
		semanticTestNote("d-recovery-two", "Recover partial GitHub push using markers", "second attempt adopted issue 42"),
	})
	if len(got) != 1 || got[0].id != "d-recovery-one" {
		t.Fatalf("canonical identity = %#v, want first marker id d-recovery-one", got)
	}
	// Simulate a partial push that published the second local record before the
	// canonical mapping was written. Recovery must recognize that alias marker.
	idx := &markerIndex{loaded: true, issues: []ghIssue{{Number: 42, Body: decisionMarker(w, "d-recovery-two")}}}
	if issue := findNoteMarker(w, idx, got[0], decisionMarker); issue != 42 {
		t.Fatalf("partial-push recovery found issue %d, want alias-marked issue 42", issue)
	}
}

// A similar operational prefix is insufficient: materially different records
// must survive independently.
func TestCanonicalNoteFilesDoNotSuppressMateriallyDifferentRecord(t *testing.T) {
	got := canonicalNoteFiles([]noteFile{
		semanticTestNote("f-dns", "GitHub push blocked by DNS", "Could not resolve github.com"),
		semanticTestNote("f-auth", "GitHub push blocked by authentication", "HTTP 401: bad credentials"),
	})
	if len(got) != 2 {
		t.Fatalf("materially different blockers collapsed to %d record(s)", len(got))
	}
}
