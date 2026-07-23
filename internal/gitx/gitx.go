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
//
// Exported (rather than const) so a test can shrink them to prove a hung
// subprocess is actually bounded without waiting out the real deadline.
var (
	LocalTimeout   = 30 * time.Second
	NetworkTimeout = 120 * time.Second
)

// Run executes git in dir under the local-operation deadline and returns
// trimmed combined output.
func Run(dir string, args ...string) (string, error) {
	return runWithTimeout(dir, LocalTimeout, args...)
}

// RunNetwork executes git in dir under the longer network-operation deadline
// — for any git child that talks to a remote (fetch, push, ls-remote).
func RunNetwork(dir string, args ...string) (string, error) {
	return runWithTimeout(dir, NetworkTimeout, args...)
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

// IsCleanExcept reports whether the working tree has no dirty TRACKED files
// OUTSIDE the given path prefixes. dacli's own .dacli workspace is dirtied by
// normal operation — closing a task rename-moves its (tracked) file between
// status folders, events and notes are written constantly — and those changes
// never participate in a code-branch merge, so a merge must tolerate them.
// Only a dirty *code* file (outside the ignored prefixes) is genuinely at risk
// of being clobbered by a merge, so only that blocks it.
func IsCleanExcept(dir string, ignore ...string) bool {
	out, err := Run(dir, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false
	}
	if out == "" {
		return true
	}
	for _, line := range strings.Split(out, "\n") {
		// porcelain v1 is "XY <path>". Run trims the whole output, so the first
		// line loses its leading space and the XY column shifts — parse by
		// trimming each line and taking the path as everything past the first
		// space, rather than by a fixed column.
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		sp := strings.IndexByte(line, ' ')
		if sp < 0 {
			continue
		}
		path := strings.TrimSpace(line[sp+1:])
		// A rename shows as "old -> new"; the destination is what matters.
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		if !underAny(path, ignore) {
			return false
		}
	}
	return true
}

func underAny(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
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
	// Tolerate a dirty .dacli — closing tasks (dacli accept, part of the ship
	// pipeline) rename-moves tracked task files, which never take part in a
	// code merge. A dirty *code* file still blocks: it could be clobbered.
	if !IsCleanExcept(root, ".dacli") {
		return nil, fmt.Errorf("working tree at %s has uncommitted code changes; commit or stash before merging", root)
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
	return RunNetwork(root, "push", "-u", "origin", branch)
}
