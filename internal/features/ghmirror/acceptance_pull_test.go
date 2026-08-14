package ghmirror

// Task 446: inbound GitHub issues may carry the human-authored acceptance
// checklist. Adoption must move that checklist into the task's canonical
// Acceptance section instead of leaving the task unverifiable.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestPullImportsAcceptanceChecklistOnce(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	body := "Background that must remain.\n\n## Acceptance criteria\n\n- [ ] first observable result\n- [x] already verified upstream\n\n## Notes\n\nKeep this too."
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
			return fmt.Sprintf(`[{"number":42,"title":"adopt this checklist","body":%q,"state":"open"}]`, body), nil
		}
		return "", nil
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdPull(ctx, []string{"core"}); err != nil {
		t.Fatalf("first pull: %v\n%s", err, out.String())
	}
	tasks, err := store.ListTasks(w, "core", "")
	if err != nil {
		t.Fatalf("list adopted tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("first pull created %d tasks, want 1", len(tasks))
	}
	got := tasks[0]
	acceptance := got.Acceptance()
	if len(acceptance) != 2 {
		t.Fatalf("canonical Acceptance has %d boxes, want 2", len(acceptance))
	}
	if acceptance[0].Text != "first observable result" || acceptance[0].Done {
		t.Fatalf("first criterion = %#v, want imported unchecked state", acceptance[0])
	}
	if acceptance[1].Text != "already verified upstream" || !acceptance[1].Done {
		t.Fatalf("second criterion = %#v, want imported checked state", acceptance[1])
	}
	contextSection, ok := got.Doc.Section("Context")
	if !ok {
		t.Fatal("adopted task has no Context section")
	}
	for _, want := range []string{"Adopted from GitHub issue #42.", "Background that must remain."} {
		if !strings.Contains(contextSection.Content, want) {
			t.Fatalf("Context omitted %q:\n%s", want, contextSection.Content)
		}
	}
	notes, ok := got.Doc.Section("Notes")
	if !ok || !strings.Contains(notes.Content, "Keep this too.") {
		t.Fatalf("issue section after acceptance was not preserved: %#v", notes)
	}
	for _, section := range got.Doc.Sections {
		if section.Title != "Acceptance" && (strings.Contains(section.Content, "first observable result") || strings.Contains(section.Content, "already verified upstream")) {
			t.Fatalf("section %q retained a second editable acceptance checklist:\n%s", section.Title, section.Content)
		}
	}

	// The imported criteria satisfy the canonical close precondition and can be
	// checked through the same store primitive used by the acceptance command.
	if !store.HasAcceptanceCriteria(got) {
		t.Fatal("adopted task still appears to have no acceptance criteria")
	}
	if newly := store.CheckAllAcceptance(got); newly != 1 {
		t.Fatalf("checking imported acceptance marked %d new boxes, want 1", newly)
	}
	if err := store.SaveTask(got); err != nil {
		t.Fatalf("save checked task: %v", err)
	}
	if err := store.MoveTask(w, got, model.StatusDone); err != nil {
		t.Fatalf("finish adopted task without an unverified escape hatch: %v", err)
	}

	// Re-pull is mapping-idempotent: it neither creates another task nor adds
	// another copy of the imported boxes.
	out.Reset()
	if err := cmdPull(ctx, []string{"core"}); err != nil {
		t.Fatalf("second pull: %v\n%s", err, out.String())
	}
	tasks, err = store.ListTasks(w, "core", "")
	if err != nil {
		t.Fatalf("list after re-pull: %v", err)
	}
	if len(tasks) != 1 || len(tasks[0].Acceptance()) != 2 {
		t.Fatalf("re-pull produced %d tasks and %d acceptance boxes, want 1 and 2", len(tasks), len(tasks[0].Acceptance()))
	}
}

func TestIssueWithoutRecognizedAcceptanceChecklistIsUnchanged(t *testing.T) {
	body := "Intro.\n\n## Done when\n\n- [ ] tempting but undocumented heading\n"
	context, boxes := issueTaskContent(ghIssue{Number: 7, Body: body})
	if len(boxes) != 0 {
		t.Fatalf("unrecognized heading invented %d canonical criteria", len(boxes))
	}
	if !strings.Contains(context, body) {
		t.Fatalf("body without a recognized checklist changed:\n%s", context)
	}
}
