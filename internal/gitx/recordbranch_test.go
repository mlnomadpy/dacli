package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func recordRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@t.t"},
		{"config", "user.name", "t"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// A product commit on trunk, plus an untracked workspace dir.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".dacli", "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".dacli", "projects", "p.md"), []byte("# p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "main.go"}, {"commit", "-qm", "feat: the product"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// The workspace record must live on its own ref, NOT on trunk. Committing it to
// trunk is what made 58% of this repo's own history bookkeeping — including one
// message repeated verbatim 61 times — which is the opposite of the engineering
// history a reader of a generated repo is looking for (dacli 193).
func TestCommitPathToBranchKeepsTrunkClean(t *testing.T) {
	dir := recordRepo(t)

	trunkBefore, _ := Run(dir, "rev-parse", "main")
	statusBefore, _ := Run(dir, "status", "--porcelain")

	sha, err := CommitPathToBranch(dir, "dacli-record", ".dacli", "record: cycle 1", "bot", "bot@agent.dacli")
	if err != nil {
		t.Fatalf("CommitPathToBranch: %v", err)
	}
	if sha == "" {
		t.Fatal("expected a commit sha")
	}

	// Trunk must not have moved.
	if trunkAfter, _ := Run(dir, "rev-parse", "main"); trunkAfter != trunkBefore {
		t.Errorf("trunk moved: %s -> %s; the record must never land on trunk", trunkBefore, trunkAfter)
	}
	// The working tree and index must be untouched — no checkout, no staging.
	if statusAfter, _ := Run(dir, "status", "--porcelain"); statusAfter != statusBefore {
		t.Errorf("working tree changed:\nbefore: %q\nafter:  %q", statusBefore, statusAfter)
	}
	// The record branch must exist and carry the workspace.
	files, err := Run(dir, "ls-tree", "-r", "--name-only", "dacli-record")
	if err != nil {
		t.Fatalf("record branch not created: %v", err)
	}
	if !strings.Contains(files, ".dacli/projects/p.md") {
		t.Errorf("record branch must contain the workspace, got:\n%s", files)
	}
	if strings.Contains(files, "main.go") {
		t.Errorf("record branch must contain ONLY the workspace, got:\n%s", files)
	}
}

// A second record must extend the branch, not orphan the first — the trajectory
// is a history, not a series of disconnected snapshots.
func TestCommitPathToBranchAppendsToHistory(t *testing.T) {
	dir := recordRepo(t)

	first, err := CommitPathToBranch(dir, "dacli-record", ".dacli", "record: cycle 1", "bot", "bot@agent.dacli")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".dacli", "projects", "q.md"), []byte("# q\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := CommitPathToBranch(dir, "dacli-record", ".dacli", "record: cycle 2", "bot", "bot@agent.dacli")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("the second record must be a new commit")
	}
	parents, _ := Run(dir, "rev-list", "--count", "dacli-record")
	if strings.TrimSpace(parents) != "2" {
		t.Errorf("record branch should have 2 commits, got %q", parents)
	}
	if parent, _ := Run(dir, "rev-parse", "dacli-record~1"); !strings.HasPrefix(parent, first[:7]) {
		t.Errorf("second record's parent = %s, want the first record %s", parent, first)
	}
}

// The record branch must keep working even when the product repo's trunk
// gitignores the whole workspace (dacli 222). A plain `git add -- .dacli`
// stages nothing when `.dacli/` is ignored, which would silently stop the
// record; the commit must force past the OUTER ignore while still honoring the
// workspace's OWN inner .dacli/.gitignore, so runs/build/worktrees never reach
// the record branch.
func TestCommitPathToBranchRecordsGitignoredWorkspace(t *testing.T) {
	dir := recordRepo(t)

	// Trunk gitignores the workspace, and the workspace ignores its own
	// regenerable subtrees.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".dacli/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".dacli", ".gitignore"), []byte("runs/\nbuild/\nworktrees/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".dacli", "runs", "r1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".dacli", "runs", "r1", "transcript.md"), []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	sha, err := CommitPathToBranch(dir, "dacli-record", ".dacli", "record: cycle 1", "bot", "bot@agent.dacli")
	if err != nil {
		t.Fatalf("CommitPathToBranch on a gitignored workspace: %v", err)
	}
	if sha == "" {
		t.Fatal("a gitignored workspace still has content to record — expected a commit")
	}

	files, err := Run(dir, "ls-tree", "-r", "--name-only", "dacli-record")
	if err != nil {
		t.Fatalf("record branch not created: %v", err)
	}
	if !strings.Contains(files, ".dacli/projects/p.md") {
		t.Errorf("record branch must carry the workspace despite the trunk ignore, got:\n%s", files)
	}
	if strings.Contains(files, ".dacli/runs/") {
		t.Errorf("record branch swept in the regenerable runs/ subtree the inner .gitignore excludes:\n%s", files)
	}
	if strings.Contains(files, "main.go") {
		t.Errorf("record branch must contain ONLY the workspace, got:\n%s", files)
	}
}

// Nothing to record is not an error, and must not create an empty commit —
// that is precisely how "record workspace after integrating 0 task(s)" ended up
// in this repo's history 61 times.
func TestCommitPathToBranchSkipsWhenUnchanged(t *testing.T) {
	dir := recordRepo(t)

	if _, err := CommitPathToBranch(dir, "dacli-record", ".dacli", "record: cycle 1", "bot", "bot@agent.dacli"); err != nil {
		t.Fatal(err)
	}
	sha, err := CommitPathToBranch(dir, "dacli-record", ".dacli", "record: cycle 2", "bot", "bot@agent.dacli")
	if err != nil {
		t.Fatalf("an unchanged workspace must not be an error: %v", err)
	}
	if sha != "" {
		t.Errorf("an unchanged workspace must not create a commit, got %s", sha)
	}
	if n, _ := Run(dir, "rev-list", "--count", "dacli-record"); strings.TrimSpace(n) != "1" {
		t.Errorf("record branch should still have 1 commit, got %q", n)
	}
}
