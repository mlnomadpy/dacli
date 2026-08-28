package ghmirror

// Task 446: inbound GitHub issues may carry the human-authored acceptance
// checklist. Adoption must move that checklist into the task's canonical
// Acceptance section instead of leaving the task unverifiable.

import (
	"fmt"
	"os"
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
	if acceptance[1].Text != "already verified upstream" || acceptance[1].Done {
		t.Fatalf("second criterion = %#v, remote checked state must not become local verification", acceptance[1])
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
	raw, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatalf("read adopted task: %v", err)
	}
	if !strings.Contains(string(raw), body) {
		t.Fatalf("adoption did not preserve the original issue body verbatim:\n%s", raw)
	}
	importRecord, ok := got.Doc.Front.GetBlock("github_acceptance_import")
	if !ok {
		t.Fatal("adopted task has no acceptance import audit record")
	}
	wantDigest := extractIssueAcceptance(body).BodyDigest
	for _, want := range []string{"issue: 42", "body_digest: " + wantDigest, "actor: a-root", "imported_at:"} {
		if !strings.Contains(importRecord, want) {
			t.Fatalf("acceptance import record omitted %q:\n%s", want, importRecord)
		}
	}

	// The imported criteria satisfy the canonical close precondition and can be
	// checked through the same store primitive used by the acceptance command.
	if !store.HasAcceptanceCriteria(got) {
		t.Fatal("adopted task still appears to have no acceptance criteria")
	}
	if newly := store.CheckAllAcceptance(got); newly != 2 {
		t.Fatalf("checking imported acceptance marked %d new boxes, want 2", newly)
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
	body := "Intro.\n\n## Done when\n\n- tempting but undocumented prose bullet\n"
	context, boxes := issueTaskContent(ghIssue{Number: 7, Body: body})
	if len(boxes) != 0 {
		t.Fatalf("unrecognized heading invented %d canonical criteria", len(boxes))
	}
	if !strings.Contains(context, body) {
		t.Fatalf("body without a recognized checklist changed:\n%s", context)
	}
}

func TestExtractIssueAcceptanceConservativeFixtures(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		criteria   []string
		ambiguous  bool
		skippedMin int
	}{
		{
			name:       "headed plain bullets and checkboxes normalize and dedupe",
			body:       "## Acceptance\n- first   observable result\n* [x] second result\n+ FIRST observable result\n",
			criteria:   []string{"first observable result", "second result"},
			skippedMin: 1,
		},
		{
			name:     "top level checkbox list without heading",
			body:     "Intro\n\n- [ ] ship Linux\n- [X] ship macOS\n- arbitrary bullet\n",
			criteria: []string{"ship Linux", "ship macOS"},
		},
		{
			name:       "code examples and non goals are excluded",
			body:       "## Examples\n- [ ] illustrative only\n\n```md\n- [ ] code only\n```\n\n## Non-goals\n- [ ] never import me\n\n## Acceptance criteria\n- real requirement\n",
			criteria:   []string{"real requirement"},
			skippedMin: 3,
		},
		{
			name:      "nested acceptance is ambiguous and fails closed",
			body:      "## Acceptance criteria\n- parent requirement\n  - [ ] nested interpretation\n",
			ambiguous: true,
		},
		{
			name:      "empty checkbox is ambiguous",
			body:      "- [ ] \n",
			ambiguous: true,
		},
		{
			name:      "numbered acceptance list is ambiguous",
			body:      "## Acceptance\n1. unclear list convention\n",
			ambiguous: true,
		},
		{
			name:       "nested standalone checklist is skipped",
			body:       "## Notes\n  - [ ] nested note\n",
			skippedMin: 1,
		},
		{
			name: "arbitrary prose bullets are not acceptance",
			body: "## Design\n- option A\n- option B\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIssueAcceptance(tt.body)
			if strings.Join(got.Criteria, "|") != strings.Join(tt.criteria, "|") {
				t.Fatalf("criteria = %#v, want %#v", got.Criteria, tt.criteria)
			}
			if (len(got.Ambiguities) > 0) != tt.ambiguous {
				t.Fatalf("ambiguities = %#v, ambiguous want %v", got.Ambiguities, tt.ambiguous)
			}
			if len(got.Skipped) < tt.skippedMin {
				t.Fatalf("skipped = %#v, want at least %d entries", got.Skipped, tt.skippedMin)
			}
			if !strings.HasPrefix(got.BodyDigest, "sha256:") || len(got.BodyDigest) != len("sha256:")+64 {
				t.Fatalf("body digest = %q, want sha256 digest", got.BodyDigest)
			}
		})
	}
}

func TestMergeAcceptanceCriteriaPreservesLocalStateAndDeduplicates(t *testing.T) {
	w := mirrorWorkspace(t)
	task, err := store.CreateTask(w, "a-root", "core", "existing task", store.TaskOpts{Accept: []string{"keep local", "same criterion"}})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	task.Doc.SetSection("Acceptance", "Local framing remains.\n- [x] keep local\n- [ ] same criterion\n")

	if added := mergeAcceptanceCriteria(task, []string{"Same   criterion", "new remote criterion", "NEW REMOTE CRITERION"}); added != 1 {
		t.Fatalf("added %d criteria, want 1", added)
	}
	sec, _ := task.Doc.Section("Acceptance")
	for _, want := range []string{"Local framing remains.", "- [x] keep local", "- [ ] same criterion", "- [ ] new remote criterion"} {
		if !strings.Contains(sec.Content, want) {
			t.Fatalf("merged Acceptance omitted %q:\n%s", want, sec.Content)
		}
	}
	if strings.Count(strings.ToLower(sec.Content), "new remote criterion") != 1 {
		t.Fatalf("new criterion was duplicated:\n%s", sec.Content)
	}
}

func TestPullPlanRefusesAmbiguousAcceptance(t *testing.T) {
	w := mirrorWorkspace(t)
	issues := []ghIssue{{Number: 42, Title: "ambiguous contract", State: "open", Body: "## Acceptance criteria\n- parent\n  - [ ] nested"}}
	plan, refused, err := planPull(w, "core", issues, map[int]bool{})
	if err != nil {
		t.Fatalf("plan pull: %v", err)
	}
	if !refused || len(plan) != 1 || plan[0].outcome != pullRefused {
		t.Fatalf("plan = %#v, refused = %v; ambiguous extraction must fail closed", plan, refused)
	}
}

func TestPullDryRunPrintsAcceptancePlanAndWritesNothing(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
			return `[{"number":42,"title":"preview contract","body":"## Acceptance criteria\n- [x] exact result\n- [ ] exact result","state":"open"}]`, nil
		}
		return "", nil
	}
	ctx, out := releaseCtx(t, w)
	if err := cmdPull(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	for _, want := range []string{"acceptance source: sha256:", `acceptance criterion: "exact result" (unchecked)`, "acceptance skipped:", "nothing was written"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("dry-run omitted %q:\n%s", want, out.String())
		}
	}
	tasks, err := store.ListTasks(w, "core", "")
	if err != nil || len(tasks) != 0 {
		t.Fatalf("dry-run wrote tasks: count=%d err=%v", len(tasks), err)
	}
}
