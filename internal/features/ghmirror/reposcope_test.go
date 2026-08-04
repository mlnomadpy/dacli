package ghmirror

import (
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// dacli 221: every gh call that writes to (or reads from) a linked project's
// repo must pass --repo explicitly. Without it, gh resolves the target from the
// workspace-root remote (cmd.Dir = w.Root), so a workspace managing several
// projects — each github-linked to a DIFFERENT repo — mirrors all but one
// project's issues into the WRONG repository. These tests pin the flag onto the
// helper surface so a future call site cannot silently drop it again.

// captureArgs swaps the stubbable gh var for one that records every invocation's
// args and returns a canned success, restoring the original on cleanup.
func captureArgs(t *testing.T, out string) *[][]string {
	t.Helper()
	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		cp := append([]string(nil), args...)
		calls = append(calls, cp)
		return out, nil
	}
	return &calls
}

// hasRepoFlag reports whether an argv carries `--repo <repo>` as an adjacent pair.
func hasRepoFlag(args []string, repo string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--repo" && args[i+1] == repo {
			return true
		}
	}
	return false
}

// ghRepo appends --repo after the subcommand verb, so gh (whose --repo is a
// per-command flag invalid at the root) accepts it — and omits it entirely when
// the repo is empty, preserving cwd resolution for the pre-link discovery paths.
func TestGHRepoAppendsRepoFlag(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := captureArgs(t, "")

	if _, err := ghRepo(w, "octo/linked", "issue", "create", "--title", "x"); err != nil {
		t.Fatalf("ghRepo: %v", err)
	}
	got := (*calls)[0]
	if got[0] != "issue" || got[1] != "create" {
		t.Fatalf("--repo must follow the subcommand verb, not precede it: %v", got)
	}
	if !hasRepoFlag(got, "octo/linked") {
		t.Fatalf("ghRepo dropped --repo octo/linked: %v", got)
	}
}

func TestGHRepoEmptyRepoOmitsFlag(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := captureArgs(t, "")

	if _, err := ghRepo(w, "", "issue", "list"); err != nil {
		t.Fatalf("ghRepo: %v", err)
	}
	for _, a := range (*calls)[0] {
		if a == "--repo" {
			t.Fatalf("an empty repo must fall back to cwd resolution (no --repo): %v", (*calls)[0])
		}
	}
}

// The regression that motivated dacli 221: the label/edit/list/view write and
// read helpers each ran a bare gh (no --repo), so they targeted the cwd remote
// instead of the linked repo. Assert every invocation they make now names the
// linked repo — the property that keeps a multi-project workspace's mirrors in
// the right repositories.
func TestWriteHelpersScopeToLinkedRepo(t *testing.T) {
	const repo = "octo/linked"
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := captureArgs(t, "[]")

	ensureLabel(w, repo, "finding")
	applyStatusLabel(w, repo, 7, model.StatusDone)
	applyTaskLabels(w, repo, 7, "area:ghmirror")
	applyFindingLabels(w, repo, 7, "severity:major", "area:ghmirror")
	// The canned "[]" is not a valid comments object, so issueComments returns a
	// parse error here (dacli 220) — irrelevant to this test, which only asserts
	// the gh call it made carries --repo. Ignore the result.
	issueComments(w, repo, 7)
	if _, _, err := fetchAllIssues(w, repo, "number,body"); err != nil {
		t.Fatalf("fetchAllIssues: %v", err)
	}

	if len(*calls) == 0 {
		t.Fatal("no gh calls captured")
	}
	for _, args := range *calls {
		if !hasRepoFlag(args, repo) {
			t.Fatalf("a mirror gh call omitted --repo %s and would hit the cwd remote: %v", repo, args)
		}
	}
}

// The marker-index snapshot (the adoption read that prevents duplicate issues)
// must also be scoped to the linked repo, or a re-push reads the cwd repo's
// markers and re-creates every issue in the linked repo as a duplicate.
func TestMarkerIndexScopesToLinkedRepo(t *testing.T) {
	const repo = "octo/linked"
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := captureArgs(t, "[]")

	idx := newMarkerIndex(w, repo)
	idx.find("<!-- dacli:whatever -->")

	if len(*calls) == 0 {
		t.Fatal("marker index made no gh call")
	}
	if !hasRepoFlag((*calls)[0], repo) {
		t.Fatalf("marker index list omitted --repo %s: %v", repo, (*calls)[0])
	}
}
