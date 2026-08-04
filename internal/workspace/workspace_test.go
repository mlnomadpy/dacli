package workspace_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func realpath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p) // macOS /var -> /private/var
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// A dacli command run from inside a LINKED git worktree must resolve to the
// MAIN worktree's .dacli, not the worktree's stale tracked copy — otherwise a
// spawned agent gets a shadow workspace and can't see its own freshly-minted
// identity or an uncommitted task (self-commit attribution + `task check`
// break). This is the fix for the worktree-shadowing class (issue #1 / #5).
func TestFindRedirectsFromLinkedWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "config", "user.email", "t@t")
	git(t, root, "config", "user.name", "t")
	if _, err := workspace.Init(root, "x"); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "init")

	wt := filepath.Join(root, ".dacli", "worktrees", "w1")
	git(t, root, "worktree", "add", "-q", wt, "-b", "feat", "HEAD")

	// From the main root: resolves to itself, no redirect.
	if w, err := workspace.Find(root); err != nil {
		t.Fatalf("Find(root): %v", err)
	} else if realpath(t, w.Root) != realpath(t, root) {
		t.Fatalf("Find(root).Root = %s; want %s", w.Root, root)
	}

	// From inside the linked worktree: redirects to the SHARED root.
	w, err := workspace.Find(wt)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := realpath(t, w.Root), realpath(t, root); got != want {
		t.Fatalf("Find(worktree).Root = %s; want shared root %s (shadow not redirected)", got, want)
	}
}

// Two tasks with the same title share a slug, so a slug-only worktree key made
// them collide onto one directory and commit onto the wrong branch (dacli 215).
// WorktreePath must disambiguate by project + seq so same-titled tasks — within
// a project (different seq) and across projects (repeated seq) — never share a
// worktree.
func TestWorktreePathDisambiguatesSameTitle(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "config", "user.email", "t@t")
	git(t, root, "config", "user.name", "t")
	w, err := workspace.Init(root, "x")
	if err != nil {
		t.Fatal(err)
	}

	// Same slug, same project, different seq — must not collide.
	a := w.WorktreePath("core", 1, "fix-thing")
	b := w.WorktreePath("core", 2, "fix-thing")
	if a == b {
		t.Fatalf("same-titled tasks share a worktree: %s", a)
	}

	// Same slug, same seq, different project — must not collide.
	c := w.WorktreePath("web", 1, "fix-thing")
	if a == c {
		t.Fatalf("cross-project same-seq tasks share a worktree: %s", a)
	}

	// The name carries project and seq so the layout stays greppable.
	if got, want := filepath.Base(a), "core-001-fix-thing"; got != want {
		t.Fatalf("WorktreePath base = %s; want %s", got, want)
	}
}

// A project slug is a single path segment. SafeSegment is the guard that keeps
// an explicit --slug or a forged --project from escaping projects/ — the
// path-traversal write/delete class (dacli 163).
func TestSafeSegment(t *testing.T) {
	ok := []string{"core", "my-project", "p1", "a-b-c-9"}
	for _, s := range ok {
		if !workspace.SafeSegment(s) {
			t.Errorf("SafeSegment(%q) = false; want true (well-formed slug)", s)
		}
	}
	bad := []string{"", ".", "..", "../x", "../../../../etc", "a/b", "/abs", "x/..", "..\\y", "foo/../bar"}
	for _, s := range bad {
		if workspace.SafeSegment(s) {
			t.Errorf("SafeSegment(%q) = true; want false (traversal/separator)", s)
		}
	}
}

// ProjectDir must never resolve a traversing slug to a path outside projects/.
// A well-formed slug resolves normally; anything that would escape is redirected
// to an in-workspace sentinel, so the operation fails safely inside .dacli.
func TestProjectDirContainment(t *testing.T) {
	root := t.TempDir()
	w, err := workspace.Init(root, "x")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := w.ProjectDir("core"), filepath.Join(w.ProjectsDir(), "core"); got != want {
		t.Fatalf("ProjectDir(core) = %s; want %s", got, want)
	}
	for _, evil := range []string{"../../../../escape", "..", "a/../../b"} {
		got := w.ProjectDir(evil)
		rel, err := filepath.Rel(w.ProjectsDir(), got)
		if err != nil || rel == ".." || filepath.IsAbs(rel) ||
			len(rel) >= 2 && rel[0] == '.' && rel[1] == '.' {
			t.Fatalf("ProjectDir(%q) = %s escaped projects/ (rel=%q)", evil, got, rel)
		}
	}
}
