package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// repoOnMainWithBranch builds a repo on main with a base commit plus a `feature`
// branch that changes code.txt, then leaves main checked out.
func repoOnMainWithBranch(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "x@x")
	git(t, dir, "config", "user.name", "x")
	git(t, dir, "checkout", "-q", "-b", "main")
	write(t, dir, "code.txt", "base\n")
	write(t, dir, ".dacli/tasks/open/1-a.md", "base task\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")

	git(t, dir, "checkout", "-q", "-b", "feature")
	write(t, dir, "code.txt", "base\nfeature line\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "feature")
	git(t, dir, "checkout", "-q", "main")
	return dir
}

// A dirty .dacli (the state `dacli accept` leaves behind by moving a tracked
// task file between status folders) must NOT block a code-branch merge — that
// is what once made `dacli ship` fail to integrate right after accepting.
func TestMergeToleratesDirtyDacli(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	// Simulate accept: move the tracked task file (dirty, tracked deletion) and
	// drop the new copy in done/ — exactly what MoveTask does.
	if err := os.Rename(filepath.Join(dir, ".dacli/tasks/open/1-a.md"), filepath.Join(dir, ".dacli/tasks/done/1-a.md")); err != nil {
		// done dir may not exist yet
		write(t, dir, ".dacli/tasks/done/1-a.md", "base task\n")
		_ = os.Remove(filepath.Join(dir, ".dacli/tasks/open/1-a.md"))
	}
	if IsClean(dir) {
		t.Fatal("precondition: tree should be dirty (.dacli task moved)")
	}
	conflicts, err := Merge(dir, "feature", "merge feature")
	if err != nil {
		t.Fatalf("merge refused despite only .dacli being dirty: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", conflicts)
	}
	if got := git(t, dir, "log", "-1", "--format=%s"); got == "" {
		t.Fatal("no merge commit")
	}
}

// A dirty CODE file still blocks the merge — it could be clobbered — and the
// failure is a real error, never a mislabeled conflict.
func TestMergeRefusesDirtyCode(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	write(t, dir, "code.txt", "base\nlocal uncommitted edit\n")
	conflicts, err := Merge(dir, "feature", "merge feature")
	if err == nil {
		t.Fatal("merge should refuse with a dirty code tree")
	}
	if len(conflicts) != 0 {
		t.Fatalf("a dirty tree is not a conflict; got conflicts %v", conflicts)
	}
}

// A merge that fails for a NON-conflict reason (here: a branch that does not
// exist) returns a real error with no conflicted files, so the caller can tell
// it apart from a genuine conflict and not swallow it to success.
func TestMergeMissingBranchIsErrorNotConflict(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	conflicts, err := Merge(dir, "does-not-exist", "merge ghost")
	if err == nil {
		t.Fatal("merging a missing branch should error")
	}
	if len(conflicts) != 0 {
		t.Fatalf("missing branch is not a conflict; got %v", conflicts)
	}
}

func TestIsCleanExcept(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	if !IsCleanExcept(dir, ".dacli") {
		t.Fatal("clean tree should be clean")
	}
	write(t, dir, ".dacli/tasks/open/2-b.md", "new\n")
	// Untracked .dacli file is invisible to --untracked-files=no, still clean.
	if !IsCleanExcept(dir, ".dacli") {
		t.Fatal(".dacli-only change should be clean-except-.dacli")
	}
	write(t, dir, "code.txt", "changed\n")
	if IsCleanExcept(dir, ".dacli") {
		t.Fatal("a dirty tracked code file must not be clean-except-.dacli")
	}
}
