// Package gitx is the shared git-operations layer — the entity-level plumbing
// that the vcs and execution slices both build on (slices never import each
// other, so shared git logic lives here). It exists for the parallel-agent
// lifecycle: isolated worktrees, push/PR, and conflict-aware merges.
package gitx

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Deadlines bound every git child so a hung subprocess (a credential-helper
// prompt, a wedged network push) can never block the caller. Under
// `dacli mcp serve` a blocked git would freeze the whole stdio loop, so this
// is a correctness property, not a nicety. Local plumbing gets a short leash;
// network operations get a longer one.
const (
	localTimeout   = 30 * time.Second
	networkTimeout = 120 * time.Second
)

// Run executes git in dir under the local-operation deadline and returns
// trimmed combined output.
func Run(dir string, args ...string) (string, error) {
	return runWithTimeout(dir, localTimeout, args...)
}

func runWithTimeout(dir string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), fmt.Errorf("git %s timed out after %s", strings.Join(args, " "), timeout)
	}
	return strings.TrimSpace(string(out)), err
}

// Available reports whether git is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// CurrentBranch names the checked-out branch (works on an unborn branch).
func CurrentBranch(dir string) string {
	b, _ := Run(dir, "branch", "--show-current")
	return b
}

// IsClean reports whether the working tree at dir has no TRACKED
// modifications. Untracked files (dacli's own gitignored .dacli/runs, or an
// as-yet-uncommitted .dacli) do not block a merge and are ignored — only
// staged/modified tracked files would actually conflict.
func IsClean(dir string) bool {
	out, err := Run(dir, "status", "--porcelain", "--untracked-files=no")
	return err == nil && out == ""
}

// BranchExists reports whether a local branch exists.
func BranchExists(dir, branch string) bool {
	_, err := Run(dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// Worktree is one entry from `git worktree list`.
type Worktree struct {
	Path   string
	Branch string
}

// AddWorktree creates an isolated worktree at path on branch (created from the
// repo's current HEAD if it does not exist) — the parallel-agent primitive:
// each agent gets its own directory and branch over the shared object store,
// so concurrent agents cannot clobber each other's working tree.
func AddWorktree(root, path, branch string) error {
	if BranchExists(root, branch) {
		if out, err := Run(root, "worktree", "add", path, branch); err != nil {
			return fmt.Errorf("worktree add: %s", out)
		}
		return nil
	}
	if out, err := Run(root, "worktree", "add", "-b", branch, path); err != nil {
		return fmt.Errorf("worktree add -b: %s", out)
	}
	return nil
}

// ListWorktrees parses `git worktree list --porcelain`.
func ListWorktrees(root string) ([]Worktree, error) {
	out, err := Run(root, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var wts []Worktree
	var cur Worktree
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if cur.Path != "" {
				wts = append(wts, cur)
			}
			cur = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		}
	}
	if cur.Path != "" {
		wts = append(wts, cur)
	}
	return wts, nil
}

// RemoveWorktree tears down a worktree (and prunes the admin entry).
func RemoveWorktree(root, path string) error {
	if out, err := Run(root, "worktree", "remove", "--force", path); err != nil {
		return fmt.Errorf("worktree remove: %s", out)
	}
	return nil
}

// Merge merges branch into the checkout at root. On conflict it ABORTS
// cleanly and returns the conflicted files — dacli never leaves a half-merged
// tree, because it cannot resolve conflicts and must not pretend to.
func Merge(root, branch, message string) (conflicts []string, err error) {
	if !IsClean(root) {
		return nil, fmt.Errorf("working tree at %s is dirty; commit or stash before merging", root)
	}
	if mergeOut, mergeErr := Run(root, "merge", "--no-ff", "-m", message, branch); mergeErr != nil {
		// Collect the conflicted files, then abort.
		diff, _ := Run(root, "diff", "--name-only", "--diff-filter=U")
		if diff != "" {
			conflicts = strings.Split(diff, "\n")
		}
		_, _ = Run(root, "merge", "--abort")
		if len(conflicts) == 0 {
			// No conflicted files means this was NOT a conflict — the merge
			// failed for another reason (missing branch, unrelated histories,
			// index lock, a timeout). Propagate the real error instead of
			// misreporting it as a conflict, which would wrongly block the task.
			detail := mergeOut
			if detail == "" {
				detail = mergeErr.Error()
			}
			return nil, fmt.Errorf("git merge %s failed: %s", branch, detail)
		}
		return conflicts, nil
	}
	return nil, nil
}

// Push pushes a branch to origin, setting upstream. Network-bound, so it gets
// the longer deadline.
func Push(root, branch string) (string, error) {
	return runWithTimeout(root, networkTimeout, "push", "-u", "origin", branch)
}
