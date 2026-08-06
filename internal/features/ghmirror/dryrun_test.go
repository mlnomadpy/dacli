package ghmirror

// Task 294: every remote-mutating github command gains --dry-run, and the
// preview must be derived from the SAME code path as the real run. These tests
// pin the two properties that make that honest: a dry-run performs ZERO writes
// (no mutating gh call, no local file mutation) and it PRINTS what it would
// create / adopt / close. Before this task the commands had no --dry-run at all,
// so `f.Reject` turned the flag into a usage error and the real path wrote — each
// test below fails against that pre-change behaviour.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"

	"bytes"
	"github.com/mlnomadpy/dacli/internal/clikit"
)

// mutatingVerbs are the leading gh verb pairs a dry-run must NEVER issue, so a
// single helper asserts "no write happened" across every command.
var mutatingVerbs = [][]string{
	{"issue", "create"}, {"issue", "edit"}, {"issue", "close"}, {"issue", "comment"},
	{"label", "create"}, {"release", "create"},
	{"project", "create"}, {"project", "field-create"}, {"project", "item-add"}, {"project", "item-edit"},
}

// assertNoWrites fails if any recorded gh call is a known mutating one. An `api
// --method POST` (the milestone create) is checked separately by callers that
// exercise it.
func assertNoWrites(t *testing.T, calls [][]string) {
	t.Helper()
	for _, v := range mutatingVerbs {
		if c := findCall(calls, v...); c != nil {
			t.Fatalf("dry-run issued a mutating gh call %v; a preview must write nothing", c)
		}
	}
	for _, c := range calls {
		if len(c) >= 3 && c[0] == "api" && c[1] == "--method" && c[2] == "POST" {
			t.Fatalf("dry-run issued a mutating gh api POST %v; a preview must write nothing", c)
		}
	}
}

// A dry-run push against a repo with a done, marker-bearing issue previews the
// adoption, the finding comment and the close — and writes nothing: no mutating
// gh call, and the local task file gains no github mapping.
func TestPushDryRunPreviewsAndWritesNothing(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	tk, err := store.CreateTask(w, "a-root", "core", "leaky thing", store.TaskOpts{})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := store.MoveTask(w, tk, model.StatusDone); err != nil {
		t.Fatalf("move done: %v", err)
	}
	// A finding about the task, so the comment preview has something to report.
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteFinding, "a real leak", store.NoteOpts{
		About:    tk.ID, // CreateNote wraps this in [[ ]]
		Severity: "major",
		Body:     "a real leak at internal/store/store.go:12",
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}

	// The remote already carries the task's issue, marker in the body, so the push
	// ADOPTS rather than creates — exercising the adopt + close + comment previews.
	body := marker(w, tk)
	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			return fmt.Sprintf(`[{"number":500,"title":%q,"body":%q,"state":"open"}]`, taskIssueTitle(tk), body), nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "view":
			return `{"comments":[]}`, nil // no comment carries the marker → the finding would post
		case len(args) >= 1 && args[0] == "api":
			return "", nil
		default:
			return "", nil
		}
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatalf("push --dry-run: %v\n%s", err, out.String())
	}

	assertNoWrites(t, calls)

	got := out.String()
	for _, want := range []string{
		"would adopt issue #500",
		"would comment on issue #500: finding",
		"would close issue #500",
		"dry-run: push would",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("push --dry-run output missing %q:\n%s", want, got)
		}
	}

	// No local write: the task must not have gained a github mapping.
	reloaded, err := store.FindTask(w, tk.ID)
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if n := mappedIssue(reloaded); n != 0 {
		t.Fatalf("dry-run wrote a github mapping (issue #%d) to the task file; a preview must not mutate local state", n)
	}
}

// A dry-run push of a NEW task (empty remote) previews the create and writes
// nothing.
func TestPushDryRunPreviewsCreate(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	tk, err := store.CreateTask(w, "a-root", "core", "brand new", store.TaskOpts{})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			return "[]", nil
		default:
			return "", nil
		}
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatalf("push --dry-run: %v\n%s", err, out.String())
	}
	assertNoWrites(t, calls)
	if !strings.Contains(out.String(), fmt.Sprintf("would create issue %q", taskIssueTitle(tk))) {
		t.Fatalf("push --dry-run did not preview the create:\n%s", out.String())
	}
}

// A dry-run release previews the cut and issues no `release create`.
func TestReleaseDryRunWritesNothing(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		// release view MISSES so a real run would create; the dry-run must still not.
		if len(args) >= 2 && args[0] == "release" && args[1] == "view" {
			return "", fmt.Errorf("release not found")
		}
		return "", nil
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdRelease(ctx, []string{"core", "v1.0.0", "--dry-run"}); err != nil {
		t.Fatalf("release --dry-run: %v", err)
	}
	assertNoWrites(t, calls)
	if !strings.Contains(out.String(), "would cut release v1.0.0 on owner/repo") {
		t.Fatalf("release --dry-run did not preview the cut:\n%s", out.String())
	}
}

// A dry-run codeowners previews the file without writing .github/CODEOWNERS.
func TestCodeownersDryRunWritesNoFile(t *testing.T) {
	// Unset, not blank: a present-but-empty token is a lost identity and is
	// refused (dacli 288). Acting as root here means having no token at all.
	t.Setenv("DACLI_AGENT", "x")
	_ = os.Unsetenv("DACLI_AGENT")
	root := t.TempDir()
	w, err := workspace.Init(root, "x")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRole(w, agentid.RootID, team.Role{
		Name: "maintainer", Summary: "builds dacli", Scope: []string{"internal/**"},
	}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	ctx := &clikit.Ctx{Stdout: &out, Stderr: &out, Cwd: root}
	if err := cmdCodeowners(ctx, []string{"--owner", "acme", "--dry-run"}); err != nil {
		t.Fatalf("codeowners --dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".github", "CODEOWNERS")); !os.IsNotExist(err) {
		t.Fatalf("codeowners --dry-run wrote CODEOWNERS; a preview must not touch the filesystem")
	}
	if !strings.Contains(out.String(), "would write") || !strings.Contains(out.String(), "internal/ @acme/maintainer") {
		t.Fatalf("codeowners --dry-run did not preview the file:\n%s", out.String())
	}
}

// A dry-run pull previews the adoption and creates no local task.
func TestPullDryRunAdoptsNothing(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
			return `[{"number":42,"title":"a human wrote this","body":"please fix","state":"open"}]`, nil
		}
		return "", nil
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdPull(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatalf("pull --dry-run: %v", err)
	}
	tasks, err := store.ListTasks(w, "core", "")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("pull --dry-run created %d task(s); a preview must adopt nothing", len(tasks))
	}
	if !strings.Contains(out.String(), "would adopt issue #42") {
		t.Fatalf("pull --dry-run did not preview the adoption:\n%s", out.String())
	}
}

// A dry-run project previews the would-create board and issues no project create.
func TestProjectDryRunWritesNothing(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "project" && args[1] == "list":
			return `{"projects":[]}`, nil // no existing board → a real run would create one
		default:
			return "", nil
		}
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdProject(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatalf("project --dry-run: %v", err)
	}
	assertNoWrites(t, calls)
	if !strings.Contains(out.String(), "would create Project v2 board") {
		t.Fatalf("project --dry-run did not preview the board:\n%s", out.String())
	}
}
