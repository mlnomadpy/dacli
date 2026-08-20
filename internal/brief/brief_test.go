package brief

import (
	"fmt"
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

func TestViewBoundarySeesFreshSiblingEvent(t *testing.T) {
	unsetAgentEnv(t)
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "p", "Task", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	old, err := LoadView(w, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, "a-sib", model.EventFinding, task.ID, "", "FRESH_BOUNDARY_EVENT"); err != nil {
		t.Fatal(err)
	}
	oldBrief, err := AssembleView(old, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(siblingsSection(t, oldBrief), "FRESH_BOUNDARY_EVENT") {
		t.Fatal("an immutable loaded view changed after a sibling append")
	}
	fresh, err := LoadView(w, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	freshBrief, err := AssembleView(fresh, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(siblingsSection(t, freshBrief), "FRESH_BOUNDARY_EVENT") {
		t.Fatal("the next loaded view did not observe the sibling append")
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

// omissionFor returns the recorded omission line whose text mentions kind, or ""
// (e.g. "findings beyond the working-memory cap").
func omissionFor(b *Brief, kind string) string {
	for _, o := range b.Omitted {
		if strings.Contains(o, kind+" beyond the working-memory cap") {
			return o
		}
	}
	return ""
}

// The findings cap must keep the most SEVERE finding even when its filename
// slug sorts last — the dacli 286 bug was that os.ReadDir handed findings back
// in alphabetical order, so a `major` finding named "zzz…" was silently dropped
// while `minor` findings named "aaa…" filled the cap. The omission must NAME
// what it dropped rather than report a bare count.
func TestFindingsCapKeepsSevereAndNamesDropped(t *testing.T) {
	t.Setenv("DACLI_AGENT", "")
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTask(w, "a-root", "p", "Task", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// MillerCap minor findings whose slugs sort BEFORE the one major finding.
	for i := 0; i < MillerCap; i++ {
		title := fmt.Sprintf("aaa minor issue %d", i)
		if _, err := store.CreateNote(w, "a-sib", "p", model.NoteFinding, title,
			store.NoteOpts{About: tk.ID, Severity: "minor", Body: "MINOR_BODY"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateNote(w, "a-sib", "p", model.NoteFinding, "zzz critical data loss",
		store.NoteOpts{About: tk.ID, Severity: "major", Body: "CRITICAL_MAJOR_BODY"}); err != nil {
		t.Fatal(err)
	}

	br, err := Assemble(w, tk.ID, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := siblingsSection(t, br); !strings.Contains(got, "CRITICAL_MAJOR_BODY") {
		t.Fatalf("the cap dropped the major finding it should have kept:\n%s", got)
	}
	om := omissionFor(br, "findings")
	if om == "" {
		t.Fatal("no findings omission was recorded")
	}
	if !strings.Contains(om, "[[f-") {
		t.Fatalf("findings omission reported a bare count, not the dropped items: %q", om)
	}
}

// The decisions cap must name what it drops, not report a bare count (dacli 286).
func TestDecisionsCapNamesDropped(t *testing.T) {
	t.Setenv("DACLI_AGENT", "")
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTask(w, "a-root", "p", "Task", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MillerCap+2; i++ {
		title := fmt.Sprintf("decision number %d", i)
		if _, err := store.CreateNote(w, "a-root", "p", model.NoteDecision, title,
			store.NoteOpts{Rejected: "the alternative", Because: "reasons", Body: "chose this"}); err != nil {
			t.Fatal(err)
		}
	}
	br, err := Assemble(w, tk.ID, Options{})
	if err != nil {
		t.Fatal(err)
	}
	om := omissionFor(br, "decisions")
	if om == "" {
		t.Fatal("no decisions omission was recorded")
	}
	if !strings.Contains(om, "[[d-") {
		t.Fatalf("decisions omission reported a bare count, not the dropped items: %q", om)
	}
}
