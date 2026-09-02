package gitx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListWorktreesReportsDetachedHeadAndLock(t *testing.T) {
	root := repoOnMainWithBranch(t)
	path := filepath.Join(t.TempDir(), "detached-worktree")
	git(t, root, "worktree", "add", "-q", "--detach", path, "main")
	git(t, root, "worktree", "lock", path)
	head, err := Run(root, "rev-parse", "main")
	if err != nil {
		t.Fatal(err)
	}
	wantHead := strings.TrimSpace(head)

	wts, err := ListWorktrees(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, wt := range wts {
		if wt.Detached {
			if wt.Branch != "" || wt.Head != wantHead || !wt.Detached || !wt.Locked {
				t.Fatalf("detached worktree metadata = %+v, want head=%s detached+locked", wt, wantHead)
			}
			return
		}
	}
	t.Fatalf("detached worktree missing from %+v", wts)
}

// Git has returned success in production after deleting the checkout but
// leaving its administrative entry marked prunable. RemoveWorktree must judge
// the re-observed registration, not the subprocess exit code (issue #647).
func TestRemoveWorktreePrunesRegistrationAfterFalseSuccess(t *testing.T) {
	root := repoOnMainWithBranch(t)
	path := filepath.Join(t.TempDir(), "feature-worktree")
	git(t, root, "worktree", "add", "-q", path, "feature")
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	wts, err := ListWorktrees(root)
	if err != nil || !worktreeRegistered(wts, path) {
		t.Fatalf("fixture did not leave a prunable registration: worktrees=%+v err=%v", wts, err)
	}

	original := runWorktreeRemove
	runWorktreeRemove = func(string, string) error { return nil }
	t.Cleanup(func() { runWorktreeRemove = original })
	if err := RemoveWorktree(root, path); err != nil {
		t.Fatal(err)
	}
	wts, err = ListWorktrees(root)
	if err != nil {
		t.Fatal(err)
	}
	if worktreeRegistered(wts, path) {
		t.Fatalf("successful remove left stale Git ownership: %+v", wts)
	}
}
