package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func worktreeGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Issue #679: a reopened task may reuse the branch from its merged generation.
// Branch ancestry proves the old landing only; pruning that checkout would
// destroy the workspace prepared for the new corrective generation.
func TestReclaimableWorktreesKeepsReopenedMergedBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	worktreeGit(t, root, "init", "-q")
	worktreeGit(t, root, "config", "user.email", "x@x")
	worktreeGit(t, root, "config", "user.name", "x")
	worktreeGit(t, root, "checkout", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "tracked"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	worktreeGit(t, root, "add", "tracked")
	worktreeGit(t, root, "commit", "-qm", "base")

	w, err := workspace.Init(root, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	task, err := CreateTask(w, "a-root", "p", "Correct merged work", TaskOpts{Accept: []string{"proof"}})
	if err != nil {
		t.Fatal(err)
	}
	branch := TaskBranch(task)
	worktreeGit(t, root, "checkout", "-qb", branch)
	if err := os.WriteFile(filepath.Join(root, "tracked"), []byte("landed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	worktreeGit(t, root, "commit", "-qam", "land task")
	worktreeGit(t, root, "checkout", "-q", "main")
	worktreeGit(t, root, "merge", "--no-ff", "-qm", "merge task", branch)
	checkout := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if err := os.MkdirAll(filepath.Dir(checkout), 0o755); err != nil {
		t.Fatal(err)
	}
	worktreeGit(t, root, "worktree", "add", "-q", checkout, branch)
	before, err := ReclaimableWorktrees(w, "main")
	if err != nil {
		t.Fatal(err)
	}
	if !containsWorktree(before, checkout) {
		t.Fatalf("fixture branch is not reclaimable before reopen: %+v", before)
	}

	CheckAllAcceptance(task)
	if err := SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := CloseTask(w, task, "a-root"); err != nil {
		t.Fatal(err)
	}
	done, err := FindTask(w, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReopenTask(w, done, "a-root", "merged implementation was incomplete"); err != nil {
		t.Fatal(err)
	}

	candidates, err := ReclaimableWorktrees(w, "main")
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if cleanPath(candidate.Path) == cleanPath(checkout) {
			t.Fatalf("reopened checkout was reclaimable solely from prior merge: %+v", candidate)
		}
	}
}

func containsWorktree(candidates []ReclaimableWorktree, path string) bool {
	for _, candidate := range candidates {
		if cleanPath(candidate.Path) == cleanPath(path) {
			return true
		}
	}
	return false
}

func TestReclaimableWorktreesIncludesOnlySafeDetachedCheckout(t *testing.T) {
	w, _, ordinary, restore := cleanupFixture(t)
	defer restore()
	detached := addDetachedCleanupWorktree(t, w, "accept-contained", "main")
	candidates, err := ReclaimableWorktrees(w, "main", ordinary)
	if err != nil {
		t.Fatal(err)
	}
	if !containsWorktree(candidates, detached) {
		t.Fatalf("clean contained detached checkout is not reclaimable: %+v", candidates)
	}
	if err := os.WriteFile(filepath.Join(detached, "scratch"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, err = ReclaimableWorktrees(w, "main", ordinary)
	if err != nil {
		t.Fatal(err)
	}
	if containsWorktree(candidates, detached) {
		t.Fatalf("dirty detached checkout became reclaimable: %+v", candidates)
	}
	if err := os.Remove(filepath.Join(detached, "scratch")); err != nil {
		t.Fatal(err)
	}
	runID := "01M14DETACHEDOWNER00000001"
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), procmon.Record{RunID: runID, Task: "task-review", Child: "a-reviewer", Claims: []string{"review"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.RunDir(runID), "worktree.txt"), []byte(detached+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, err = ReclaimableWorktrees(w, "main", ordinary)
	if err != nil {
		t.Fatal(err)
	}
	if containsWorktree(candidates, detached) {
		t.Fatalf("live-owned detached checkout became reclaimable: %+v", candidates)
	}
}
