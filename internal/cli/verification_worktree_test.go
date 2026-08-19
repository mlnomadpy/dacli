package cli

import (
	"os"
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

	// The files are an observable execution-location probe, not merely a Git
	// provenance assertion: a root substitution makes these appear in root.
	run(t, linked, 0, "task", "check", "001", "--n", "1", "--verify", "pwd > checked-from.txt")
	run(t, linked, 0, "accept", "002", "--allow-unlanded", "--verify", "pwd > accepted-from.txt")
	for _, name := range []string{"checked-from.txt", "accepted-from.txt"} {
		got, err := os.ReadFile(filepath.Join(linked, name))
		if err != nil {
			t.Fatalf("verification did not run in linked worktree (%s): %v", name, err)
		}
		if strings.TrimSpace(string(got)) != linked {
			t.Fatalf("verification %s ran in %q, want linked worktree %q", name, strings.TrimSpace(string(got)), linked)
		}
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("verification %s leaked into main workspace: %v", name, err)
		}
	}

	// Read through root to prove persistence used the shared workspace rather
	// than the linked worktree's stale .dacli snapshot.
	for _, seq := range []string{"001", "002"} {
		records := store.VerificationEvidenceRecords(findTaskDoc(t, root, seq))
		if len(records) != 1 {
			t.Fatalf("task %s verification records = %#v, want one", seq, records)
		}
		if got := records[0]; got.Branch != branch || got.CommitSHA != head {
			t.Fatalf("task %s provenance = branch %q sha %q, want linked branch %q sha %q", seq, got.Branch, got.CommitSHA, branch, head)
		}
	}
}
