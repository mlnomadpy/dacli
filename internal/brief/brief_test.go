package brief

import (
	"os"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// unsetAgentEnv clears DACLI_AGENT for the test, restoring whatever the
// process started with. t.Setenv cannot unset a variable, and since dacli 288
// a present-but-empty DACLI_AGENT is a lost token that fails closed rather
// than resolving to root — so a test wanting the root identity must remove
// the variable entirely, not blank it.
func unsetAgentEnv(t *testing.T) {
	t.Helper()
	if v, ok := os.LookupEnv("DACLI_AGENT"); ok {
		t.Setenv("DACLI_AGENT", v)
		_ = os.Unsetenv("DACLI_AGENT")
	}
}

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
	unsetAgentEnv(t)
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
	unsetAgentEnv(t)
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
