package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// The parallel lifecycle: two tasks each get an ISOLATED worktree+branch,
// commits land on their own branches without touching main or each other,
// and clean branches merge back. This is what makes --parallel real.
func TestParallelWorktreeLifecycle(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	writeAt(t, dir, "base.txt", "shared base\n")
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-q", "-m", "base")

	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Feature A", "--project", "p", "--accept", "a")
	run(t, dir, 0, "task", "add", "Feature B", "--project", "p", "--accept", "b")

	// Two isolated worktrees — different directories, different branches.
	run(t, dir, 0, "worktree", "add", "--task", "feature-a")
	run(t, dir, 0, "worktree", "add", "--task", "feature-b")
	list := run(t, dir, 0, "worktree", "list")
	if !strings.Contains(list, "dacli/001-feature-a") || !strings.Contains(list, "dacli/002-feature-b") {
		t.Fatalf("both worktrees not listed:\n%s", list)
	}
	wtA := filepath.Join(dir, ".dacli", "worktrees", "feature-a")
	wtB := filepath.Join(dir, ".dacli", "worktrees", "feature-b")

	// Each "agent" works IN ITS OWN worktree on non-overlapping files, and
	// commits via dacli commit (as root, rw) on its branch.
	writeAt(t, wtA, "a.txt", "feature A work\n")
	run(t, wtA, 0, "commit", "001: add feature A")
	writeAt(t, wtB, "b.txt", "feature B work\n")
	run(t, wtB, 0, "commit", "002: add feature B")

	// main is untouched by either — true isolation.
	if _, err := exec.Command("test", "-f", filepath.Join(dir, "a.txt")).Output(); err == nil {
		t.Error("feature A leaked into main's working tree")
	}
	if branch := gitAt(t, dir, "branch", "--show-current"); branch != "main" {
		t.Errorf("main checkout moved: %s", branch)
	}

	// Mark both tasks done (integrate merges done tasks' branches).
	for _, slug := range []string{"feature-a", "feature-b"} {
		run(t, dir, 0, "task", "check", slug, "--all")
		run(t, dir, 0, "task", "done", slug)
	}

	// Both clean branches integrate — serialized, in order.
	out := run(t, dir, 0, "integrate")
	if !strings.Contains(out, "integrated 2 branch(es)") {
		t.Fatalf("integrate did not merge both:\n%s", out)
	}
	// Both files now on main.
	if gitAt(t, dir, "log", "--oneline", "--all") == "" {
		t.Fatal("no commits")
	}
	files := gitAt(t, dir, "ls-files")
	if !strings.Contains(files, "a.txt") || !strings.Contains(files, "b.txt") {
		t.Errorf("integrated files missing from main: %s", files)
	}
}

// A merge conflict does NOT half-merge: it aborts, blocks the task, and files
// a finding — because dacli cannot resolve conflicts and must not pretend to.
func TestMergeConflictBlocksNotBreaks(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	writeAt(t, dir, "shared.txt", "original line\n")
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-q", "-m", "base")

	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Edit shared", "--project", "p", "--accept", "a")
	run(t, dir, 0, "worktree", "add", "--task", "edit-shared")
	wt := filepath.Join(dir, ".dacli", "worktrees", "edit-shared")

	// The branch edits shared.txt...
	writeAt(t, wt, "shared.txt", "branch line\n")
	run(t, wt, 0, "commit", "001: edit shared on the branch")
	// ...and main edits the SAME line differently → conflict on merge.
	writeAt(t, dir, "shared.txt", "main line\n")
	gitAt(t, dir, "commit", "-aqm", "main edits shared")

	refusal := run(t, dir, 3, "merge", "--task", "edit-shared")
	if !strings.Contains(refusal, "merge conflict") || !strings.Contains(refusal, "shared.txt") || !strings.Contains(refusal, "nothing was half-merged") {
		t.Fatalf("conflict not surfaced as a clean refusal:\n%s", refusal)
	}
	// main has no tracked modifications — the merge was aborted, not left
	// half-done (untracked .dacli is expected and irrelevant to the merge).
	if status := gitAt(t, dir, "status", "--porcelain", "--untracked-files=no"); status != "" {
		t.Errorf("merge left tracked changes behind: %q", status)
	}
	// And no MERGE_HEAD lingers (a clean abort, not an in-progress merge).
	if _, err := exec.Command("git", "-C", dir, "rev-parse", "--verify", "MERGE_HEAD").Output(); err == nil {
		t.Error("merge left MERGE_HEAD — not cleanly aborted")
	}
	// The task is blocked, with the conflict recorded as an event.
	if blocked := run(t, dir, 0, "task", "list", "--status", "blocked"); !strings.Contains(blocked, "edit-shared") {
		t.Errorf("task not blocked on conflict:\n%s", blocked)
	}
	if events := run(t, dir, 0, "events", "tail"); !strings.Contains(events, "block") {
		t.Errorf("conflict not recorded as a block event:\n%s", events)
	}
}
