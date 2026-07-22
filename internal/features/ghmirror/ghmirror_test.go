package ghmirror

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func mirrorWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	return w
}

// G1: exactly one status label is applied and the other three are stripped, so
// a re-push never stacks a second status: label on the same issue.
func TestStatusLabelDedup(t *testing.T) {
	for _, s := range model.AllStatuses {
		add := statusLabel(s)
		if add != "status:"+string(s) {
			t.Fatalf("statusLabel(%s) = %q", s, add)
		}
		others := otherStatusLabels(s)
		if len(others) != len(model.AllStatuses)-1 {
			t.Fatalf("otherStatusLabels(%s): got %d, want %d", s, len(others), len(model.AllStatuses)-1)
		}
		for _, o := range others {
			if o == add {
				t.Fatalf("otherStatusLabels(%s) must not include the applied label %q", s, add)
			}
			if !strings.HasPrefix(o, "status:") {
				t.Fatalf("stale label %q is not a status: label", o)
			}
		}
	}
}

// G2: the decision marker is keyed on BOTH the note id and the workspace id, and
// is distinct from the task marker so the two mirrors never adopt each other.
func TestDecisionMarkerKeying(t *testing.T) {
	w := mirrorWorkspace(t)
	mk := decisionMarker(w, "d-example")
	if !strings.Contains(mk, "d-example") {
		t.Fatalf("marker %q omits the note id", mk)
	}
	if !strings.Contains(mk, "ws:"+w.ID) {
		t.Fatalf("marker %q omits the workspace id", mk)
	}
	// A task marker for the same id must NOT be a substring of the decision
	// marker (searchByMarker matches by substring), or adoption would cross.
	tk := &store.Task{ID: "d-example"}
	if strings.Contains(mk, marker(w, tk)) {
		t.Fatalf("decision marker %q collides with task marker %q", mk, marker(w, tk))
	}
}

// G2: a decision note read from disk carries no issue mapping (create runs),
// and once the mapping is written back, mappedIssueDoc reports it — the local
// half of the idempotency guarantee (a second push skips create).
func TestDecisionMappingIdempotency(t *testing.T) {
	w := mirrorWorkspace(t)
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteDecision, "use labeled issues", store.NoteOpts{
		Rejected: "GraphQL Discussions",
		Because:  "reuses the existing marker idempotency machinery",
		Body:     "mirror decisions as issues labeled decision",
	}); err != nil {
		t.Fatalf("create decision: %v", err)
	}

	notes, err := decisionNotes(w, "core")
	if err != nil {
		t.Fatalf("decisionNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d decision notes, want 1", len(notes))
	}
	dn := notes[0]
	if dn.id == "" {
		t.Fatalf("decision note has no id")
	}
	if got := mappedIssueDoc(dn.doc); got != 0 {
		t.Fatalf("unmapped note reports issue %d, want 0", got)
	}

	// The WHY must survive into the issue body.
	body := decisionBody(w, dn)
	for _, want := range []string{"GraphQL Discussions", "reuses the existing marker idempotency", "use labeled issues", dn.id} {
		if !strings.Contains(body, want) {
			t.Fatalf("decision body missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, decisionMarker(w, dn.id)) {
		t.Fatalf("decision body missing its marker")
	}

	// Simulate the write-back and re-read: the second push must see the mapping.
	dn.doc.Front.SetBlock("github", "  issue: 42\n  repo: owner/repo")
	if err := mdstore.WriteFile(dn.path, dn.doc); err != nil {
		t.Fatalf("write back: %v", err)
	}
	reread, err := decisionNotes(w, "core")
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if got := mappedIssueDoc(reread[0].doc); got != 42 {
		t.Fatalf("mapped note reports issue %d, want 42", got)
	}
}

// decisionNotes on a project with no decisions dir is empty, not an error.
func TestDecisionNotesEmpty(t *testing.T) {
	w := mirrorWorkspace(t)
	notes, err := decisionNotes(w, "core")
	if err != nil {
		t.Fatalf("decisionNotes on empty project: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("got %d notes, want 0", len(notes))
	}
}
