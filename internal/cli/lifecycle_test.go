package cli

import (
	"os"
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
	wtA := filepath.Join(dir, ".dacli", "worktrees", "p-001-feature-a")
	wtB := filepath.Join(dir, ".dacli", "worktrees", "p-002-feature-b")

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
	wt := filepath.Join(dir, ".dacli", "worktrees", "p-001-edit-shared")

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

// worktree prune reclaims the checkouts that pile up one-per-task: a merged
// branch and a finished (done) run are both swept, while a live agent's
// worktree — whether it has committed work or is a bare just-spawned tip — is
// left untouched. Without a prune these accumulate to gigabytes (dacli 252).
func TestWorktreePruneReclaimsMergedAndFinished(t *testing.T) {
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
	run(t, dir, 0, "task", "add", "Merged feature", "--project", "p", "--accept", "a")   // 001
	run(t, dir, 0, "task", "add", "Finished feature", "--project", "p", "--accept", "a") // 002
	run(t, dir, 0, "task", "add", "Live feature", "--project", "p", "--accept", "a")     // 003
	run(t, dir, 0, "task", "add", "Bare feature", "--project", "p", "--accept", "a")     // 004

	for _, slug := range []string{"merged-feature", "finished-feature", "live-feature", "bare-feature"} {
		run(t, dir, 0, "worktree", "add", "--task", slug)
	}
	wtMerged := filepath.Join(dir, ".dacli", "worktrees", "p-001-merged-feature")
	wtFinished := filepath.Join(dir, ".dacli", "worktrees", "p-002-finished-feature")
	wtLive := filepath.Join(dir, ".dacli", "worktrees", "p-003-live-feature")
	wtBare := filepath.Join(dir, ".dacli", "worktrees", "p-004-bare-feature")

	// 001: commit, then land the branch on main with a --no-ff merge done
	// DIRECTLY in git — not via `dacli integrate`, which would clean up its own
	// worktree. This is the real leak: an async GitHub auto-merge (or a crashed
	// prior loop) lands the branch while its worktree is never torn down.
	writeAt(t, wtMerged, "m.txt", "merged work\n")
	run(t, wtMerged, 0, "commit", "001: merged work")
	gitAt(t, dir, "merge", "--no-ff", "-m", "land 001", "dacli/001-merged-feature")

	// 002: commit and mark done but never integrate → run finished, unmerged.
	writeAt(t, wtFinished, "f.txt", "finished work\n")
	run(t, wtFinished, 0, "commit", "002: finished work")
	run(t, dir, 0, "task", "check", "finished-feature", "--all")
	run(t, dir, 0, "task", "done", "finished-feature")

	// 003: commit, still active → a live agent mid-work.
	writeAt(t, wtLive, "l.txt", "live work\n")
	run(t, wtLive, 0, "commit", "003: live work")

	// 004: just spawned, zero commits, still active → a bare-tipped live spawn.

	// dry-run reports what would go but removes nothing.
	preview := run(t, dir, 0, "worktree", "prune", "--dry-run")
	if !strings.Contains(preview, "merged-feature") || !strings.Contains(preview, "finished-feature") {
		t.Fatalf("dry-run did not name the reclaimable worktrees:\n%s", preview)
	}
	if strings.Contains(preview, "live-feature") || strings.Contains(preview, "bare-feature") {
		t.Fatalf("dry-run flagged a live worktree for pruning:\n%s", preview)
	}
	if _, err := os.Stat(wtMerged); err != nil {
		t.Fatalf("dry-run removed the merged worktree: %v", err)
	}

	// The real prune reclaims only the merged + finished checkouts.
	out := run(t, dir, 0, "worktree", "prune")
	if !strings.Contains(out, "reclaimed 2 worktree(s)") {
		t.Fatalf("prune did not reclaim the two dead worktrees:\n%s", out)
	}
	if _, err := os.Stat(wtMerged); !os.IsNotExist(err) {
		t.Errorf("merged worktree not reclaimed (stat err %v)", err)
	}
	if _, err := os.Stat(wtFinished); !os.IsNotExist(err) {
		t.Errorf("finished-run worktree not reclaimed (stat err %v)", err)
	}
	if _, err := os.Stat(wtLive); err != nil {
		t.Errorf("live worktree with committed work was reclaimed: %v", err)
	}
	if _, err := os.Stat(wtBare); err != nil {
		t.Errorf("bare live spawn was reclaimed: %v", err)
	}

	// A merged branch is deleted with its worktree; a finished-but-unmerged
	// branch is kept so the un-landed fix is never lost.
	branches := gitAt(t, dir, "branch", "--list")
	if strings.Contains(branches, "dacli/001-merged-feature") {
		t.Errorf("merged branch not deleted:\n%s", branches)
	}
	if !strings.Contains(branches, "dacli/002-finished-feature") {
		t.Errorf("finished-but-unmerged branch was deleted, losing the fix:\n%s", branches)
	}
}
