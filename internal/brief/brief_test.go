package brief

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// siblingsSection returns the rendered "What siblings found" section content,
// or "" when the brief omitted it.
func siblingsSection(t *testing.T, b *Brief) string {
	t.Helper()
	for _, s := range b.Sections {
		if s.Title == "What siblings found" {
			return s.Content
		}
	}
	return ""
}

// A pending finding event about a SIBLING task in the same project must show in
// this task's brief exactly as a materialized note about that task would — the
// scope of the two feeds now matches, so a finding's brief visibility no longer
// flips when the owner syncs it into a note (issue #21).
func TestSiblingsPendingEventsAreProjectScoped(t *testing.T) {
	t.Setenv("DACLI_AGENT", "")
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	a, err := store.CreateTask(w, "a-root", "p", "Task A", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.CreateTask(w, "a-root", "p", "Task B", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// A finding filed against sibling task A, still pending (not synced).
	if _, err := eventlog.Append(w, "a-sib", model.EventFinding, a.ID, "", "SIBLING_FINDING_ABOUT_A"); err != nil {
		t.Fatal(err)
	}

	br, err := Assemble(w, b.ID, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := siblingsSection(t, br); !strings.Contains(got, "SIBLING_FINDING_ABOUT_A") {
		t.Fatalf("task B brief hid a same-project sibling's pending finding:\n%s", got)
	}
}

// A pending finding event about a task in a DIFFERENT project must NOT leak into
// this project's brief — the finding NOTES feed is per-project (store.ListNotes),
// and the events feed now matches that scope.
func TestSiblingsPendingEventsDoNotCrossProject(t *testing.T) {
	t.Setenv("DACLI_AGENT", "")
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Q", "q", "g", ""); err != nil {
		t.Fatal(err)
	}
	b, err := store.CreateTask(w, "a-root", "p", "Task B", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateTask(w, "a-root", "q", "Other", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, "a-sib", model.EventFinding, other.ID, "", "FINDING_IN_OTHER_PROJECT"); err != nil {
		t.Fatal(err)
	}

	br, err := Assemble(w, b.ID, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := siblingsSection(t, br); strings.Contains(got, "FINDING_IN_OTHER_PROJECT") {
		t.Fatalf("task B brief leaked a finding from another project:\n%s", got)
	}
}
