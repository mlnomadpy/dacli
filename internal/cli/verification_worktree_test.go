package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Verification must use the caller's checkout while task records continue to
// live in the shared main workspace. workspace.Find deliberately redirects a
// linked worktree to that main workspace, so using Workspace.Root here would
// both execute and attest the wrong artifact (issue #693).
func TestVerificationUsesCallerLinkedWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	gitRepo(t, root)
	run(t, root, 0, "init", "--name", "verification")
	run(t, root, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, root, 0, "task", "add", "Check from linked tree", "--project", "p", "--accept", "`pwd` is the linked worktree")
	run(t, root, 0, "task", "add", "Accept from linked tree", "--project", "p", "--accept", "work is accepted", "--force")

	// A linked worktree contains a frozen .dacli snapshot. Commands invoked
	// there must write task state through the shared main workspace instead.
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "workspace snapshot")
	linked := filepath.Join(root, ".dacli", "worktrees", "verification-provenance")
	gitIn(t, root, "worktree", "add", "-q", "-b", "verification-provenance", linked, "HEAD")
	gitIn(t, linked, "commit", "--allow-empty", "-q", "-m", "linked artifact")
	branch := strings.TrimSpace(gitIn(t, linked, "branch", "--show-current"))
	head := strings.TrimSpace(gitIn(t, linked, "rev-parse", "HEAD"))
	w, err := workspace.Find(linked)
	if err != nil {
		t.Fatal(err)
	}
	if w.Root != root {
		t.Fatalf("linked worktree resolved workspace root %q, want shared root %q", w.Root, root)
	}

	// Assert execution location without changing the tree being certified. The
	// old `pwd > file` probe proved cwd by making acceptance evidence invalid:
	// the command itself dirtied the verified worktree.
	verifyCWD := "test \"$(pwd)\" = \"" + linked + "\""
	run(t, linked, 0, "task", "check", "001", "--n", "1", "--verify", verifyCWD)
	run(t, linked, 0, "accept", "002", "--allow-unlanded", "--verify", verifyCWD)

	// Read through root to prove persistence used the shared workspace rather
	// than the linked worktree's stale .dacli snapshot.
	for _, seq := range []string{"001", "002"} {
		records := store.VerificationEvidenceRecords(findTaskDoc(t, root, seq))
		if len(records) != 1 {
			t.Fatalf("task %s verification records = %#v, want one", seq, records)
		}
		if got := records[0]; got.Branch != branch || got.CommitSHA != head || got.WorkingDirectory != linked {
			t.Fatalf("task %s provenance = branch %q sha %q cwd %q, want linked branch %q sha %q cwd %q", seq, got.Branch, got.CommitSHA, got.WorkingDirectory, branch, head, linked)
		}
	}
}
