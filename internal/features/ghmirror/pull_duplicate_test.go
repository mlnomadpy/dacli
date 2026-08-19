package ghmirror

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func stubPullIssues(t *testing.T, json string) {
	t.Helper()
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
			return json, nil
		}
		return "", nil
	}
}

func TestPullDryRunReportsEveryReconciliationOutcomeWithoutWriting(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	exact, err := store.CreateTask(w, "a-root", "core", "Normalize exact title", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	possible, err := store.CreateTask(w, "a-root", "core", "Prevent duplicate GitHub issue adoption", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	mapped, err := store.CreateTask(w, "a-root", "core", "already linked", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	mapped.Doc.Front.SetBlock("github", githubBlock(3, "owner/repo"))
	if err := store.SaveTask(mapped); err != nil {
		t.Fatal(err)
	}

	stubPullIssues(t, fmt.Sprintf(`[
		{"number":1,"title":"brand new unrelated work","body":"","state":"open"},
		{"number":2,"title":"NORMALIZE EXACT TITLE!","body":"","state":"open"},
		{"number":3,"title":"already linked","body":"","state":"open"},
		{"number":4,"title":"Prevent semantic duplicate issue adoption","body":"","state":"open"},
		{"number":5,"title":"mirrored","body":%q,"state":"open"}
	]`, marker(w, &store.Task{ID: "t-remote"})))

	ctx, out := releaseCtx(t, w)
	if err := cmdPull(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out.String())
	}
	for _, want := range []string{"issue #1: create", "issue #2: exact-match/link", "issue #3: already-mapped", "issue #4: possible-duplicate", "issue #5: refused", "nothing was written"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("dry-run missing %q:\n%s", want, out.String())
		}
	}
	if mappedIssue(exact) != 0 || mappedIssue(possible) != 0 {
		t.Fatal("dry-run mutated a duplicate candidate")
	}
	tasks, _ := store.ListTasks(w, "core", "")
	if len(tasks) != 3 {
		t.Fatalf("dry-run created a task: got %d, want 3", len(tasks))
	}
}

func TestPullLinksNormalizedExactMatchAndRepeatedPullIsIdempotent(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	existing, err := store.CreateTask(w, "a-root", "core", "Normalize exact title", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	stubPullIssues(t, `[{"number":42,"title":"NORMALIZE EXACT TITLE!","body":"","state":"open"}]`)
	ctx, out := releaseCtx(t, w)
	if err := cmdPull(ctx, []string{"core"}); err != nil {
		t.Fatalf("first pull: %v\n%s", err, out.String())
	}
	reloaded, _ := store.FindTask(w, existing.ID)
	if mappedIssue(reloaded) != 42 {
		t.Fatalf("exact match mapped issue #%d, want 42", mappedIssue(reloaded))
	}
	if err := cmdPull(ctx, []string{"core"}); err != nil {
		t.Fatalf("second pull: %v", err)
	}
	tasks, _ := store.ListTasks(w, "core", "")
	if len(tasks) != 1 {
		t.Fatalf("repeated pull produced %d tasks, want 1", len(tasks))
	}
}

func TestPullPossibleDuplicateRefusesBeforeCreatingUnrelatedIssue(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	if _, err := store.CreateTask(w, "a-root", "core", "Prevent duplicate GitHub issue adoption", store.TaskOpts{}); err != nil {
		t.Fatal(err)
	}
	stubPullIssues(t, `[
		{"number":10,"title":"completely unrelated new task","body":"","state":"open"},
		{"number":11,"title":"Prevent semantic duplicate issue adoption","body":"","state":"open"}
	]`)
	ctx, _ := releaseCtx(t, w)
	err := cmdPull(ctx, []string{"core"})
	if err == nil || !strings.Contains(err.Error(), "resolve it explicitly") {
		t.Fatalf("possible duplicate error = %v, want actionable refusal", err)
	}
	tasks, _ := store.ListTasks(w, "core", "")
	if len(tasks) != 1 {
		t.Fatalf("refused pull partially created unrelated issue: %d tasks", len(tasks))
	}
}

func TestPullRefusesTwoIssuesClaimingOneExactTaskMapping(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	if _, err := store.CreateTask(w, "a-root", "core", "One canonical task", store.TaskOpts{}); err != nil {
		t.Fatal(err)
	}
	stubPullIssues(t, `[
		{"number":12,"title":"ONE CANONICAL TASK!","body":"","state":"open"},
		{"number":13,"title":"One canonical task","body":"","state":"open"}
	]`)
	ctx, _ := releaseCtx(t, w)
	err := cmdPull(ctx, []string{"core"})
	if err == nil || !strings.Contains(err.Error(), "resolve it explicitly") {
		t.Fatalf("competing exact mappings error = %v, want refusal", err)
	}
	tasks, _ := store.ListTasks(w, "core", "")
	if len(tasks) != 1 || mappedIssue(tasks[0]) != 0 {
		t.Fatalf("conflicting exact pull mutated mapping: tasks=%d mapping=%d", len(tasks), mappedIssue(tasks[0]))
	}
}

func TestPullDuplicateScopeIgnoresDoneAndOtherProjects(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	done, err := store.CreateTask(w, "a-root", "core", "Reusable completed title", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, done, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "other", "Other", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(w, "a-root", "other", "Cross project title", store.TaskOpts{}); err != nil {
		t.Fatal(err)
	}
	stubPullIssues(t, `[
		{"number":20,"title":"Reusable completed title","body":"","state":"open"},
		{"number":21,"title":"Cross project title","body":"","state":"open"}
	]`)
	ctx, out := releaseCtx(t, w)
	if err := cmdPull(ctx, []string{"core"}); err != nil {
		t.Fatalf("pull: %v\n%s", err, out.String())
	}
	tasks, _ := store.ListTasks(w, "core", "")
	if len(tasks) != 3 {
		t.Fatalf("done/cross-project candidates blocked creation: got %d core tasks, want 3", len(tasks))
	}
}
