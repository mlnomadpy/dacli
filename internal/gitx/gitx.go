// Package gitx is the shared git-operations layer — the entity-level plumbing
// that the vcs and execution slices both build on (slices never import each
// other, so shared git logic lives here). It exists for the parallel-agent
// lifecycle: isolated worktrees, push/PR, and conflict-aware merges.
package gitx

import (
	"context"
	"errors"
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
// WaitDelay bounds how long the call waits for OUTPUT after the deadline fired
// and the git process was killed. Killing git does not close the output pipe a
// grandchild inherited — a credential helper git forked, a spawned ssh — and
// CombinedOutput blocks until every writer closes it, so without this the
// deadline bounded the child but not the call, and the correctness property
// above did not actually hold (dacli 213). execution sets the same knob on
// agent children for the same reason; this closes the inconsistency.
//
// Exported (rather than const) so a test can shrink them to prove a hung
// subprocess is actually bounded without waiting out the real deadline.
var (
	LocalTimeout   = 30 * time.Second
	NetworkTimeout = 120 * time.Second
	WaitDelay      = 5 * time.Second
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
	// Give up on the output pipes shortly after the kill, so a grandchild still
	// holding them open cannot stretch the call past its deadline (dacli 213).
	cmd.WaitDelay = WaitDelay
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

// DirtyPaths returns every path `git status --porcelain` reports dirty at
// dir — tracked and untracked alike — excluding paths under the given
// prefixes. Unlike IsClean/IsCleanExcept, which only report clean-or-not,
// this names WHAT is dirty: a worktree-escape guard needs the paths
// themselves to revert them, not just a verdict. --untracked-files=all keeps
// git from collapsing a brand-new untracked directory into one line for its
// own name, so a stray file two levels down is named, not its grandparent.
func DirtyPaths(dir string, ignore ...string) ([]string, error) {
	out, err := Run(dir, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
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
			paths = append(paths, path)
		}
	}
	return paths, nil
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
	return RunNetwork(root, "push", "-u", "origin", "--", branch)
}

// requireCheckedOut refuses when `branch` is not what HEAD points at. A
// `merge --ff-only origin/<branch>` or `rebase origin/<branch>` acts on the
// CHECKOUT, never on the named branch, so running one from the wrong branch
// silently rewrites the wrong history — a loop started from a feature branch
// that happens to be an ancestor of origin/main would have syncTrunk
// fast-forward THAT branch onto trunk (dacli 214). Refusing loudly is the only
// safe reading: gitx cannot know whether the caller meant to switch branches.
// worktreeFor returns the path of the worktree that has `branch` checked out,
// or "" when no linked worktree does. It is what lets a rebase happen in the
// tree that actually holds the branch rather than the one that merely knows
// about it.
func worktreeFor(root, branch string) string {
	wts, err := ListWorktrees(root)
	if err != nil {
		return ""
	}
	for _, wt := range wts {
		if wt.Branch == branch && wt.Path != root {
			return wt.Path
		}
	}
	return ""
}

func requireCheckedOut(root, branch, op string) error {
	cur := CurrentBranch(root)
	if cur == branch {
		return nil
	}
	if cur == "" {
		return fmt.Errorf("%s %s: HEAD is detached (or on an unborn branch) at %s, not on %s — checkout %s first", op, branch, root, branch, branch)
	}
	return fmt.Errorf("%s %s: %s is on branch %s, not %s — refusing to operate on the wrong branch", op, branch, root, cur, branch)
}

// FastForward fetches origin and fast-forwards the LOCAL `branch` (which it
// verifies IS the currently checked-out branch — dacli 214) up to
// origin/<branch>. It exists for a
// checkout whose remote gained commits the local clone does not have yet —
// the case a `dacli loop --pr --auto` run hits once GitHub merges a fixer's
// PR asynchronously and deletes its branch, leaving local main stale. It
// never discards local work: --ff-only refuses (and this returns that
// refusal) the moment local has a commit origin lacks, rather than
// force-syncing over it — the caller decides what to do next (retry a
// rebase, or just log and move on).
func FastForward(root, branch string) (string, error) {
	// Checked BEFORE the fetch: a refusal must not have side effects.
	if err := requireCheckedOut(root, branch, "fast-forward"); err != nil {
		return err.Error(), err
	}
	// `--` terminates options so a branch value can never be read as a git flag
	// (e.g. --upload-pack=<cmd>). Defense in depth: in-repo callers pass safe
	// dacli/<n> names, but the separator makes the guarantee local to gitx.
	if out, err := RunNetwork(root, "fetch", "-q", "origin", "--", branch); err != nil {
		return out, fmt.Errorf("fetch origin %s: %s", branch, out)
	}
	return Run(root, "merge", "--ff-only", "origin/"+branch)
}

// PushSync pushes branch to origin, and on a non-fast-forward rejection —
// origin gained a commit the local checkout does not have since it was last
// synced, e.g. an async `gh pr merge --auto` landed on this same branch —
// fetches and rebases the local branch onto origin/<branch>, then retries the
// push once. A rebase conflict aborts cleanly (never leaves the tree
// mid-rebase); the returned string is always the full diagnostic (existing
// callers just print it, e.g. `fmt.Errorf("push failed: %s", out)`), so a
// synced-retry failure is exactly as visible as a plain push failure.
//
// The push itself is branch-agnostic and stays that way — `dacli push --task N`
// legitimately pushes a task branch from a root checkout sitting on trunk, and
// a push touches no working tree. The REBASE fallback does not have that
// freedom: `rebase origin/<branch>` rewrites whatever is checked out, so it is
// gated on branch actually being the checkout and refuses otherwise rather than
// rewriting an unrelated branch's history (dacli 214).
func PushSync(root, branch string) (string, error) {
	out, err := Push(root, branch)
	if err == nil || !isNonFastForward(out) {
		return out, err
	}
	// Rebase where the branch actually IS. A worktree agent's branch is checked
	// out in its own linked worktree, never in `root` — root is the shared main
	// checkout, whose HEAD is trunk — so requireCheckedOut always failed here
	// and PushSync could never rebase a worktree child's branch at all. The
	// push then died, `dacli pr` never ran, and the branch sat there looking
	// abandoned while its work was fine.
	if wt := worktreeFor(root, branch); wt != "" {
		root = wt
	}
	if cerr := requireCheckedOut(root, branch, "push --sync"); cerr != nil {
		detail := fmt.Sprintf("push rejected (non-fast-forward) and cannot rebase: %s — original: %s", cerr, out)
		return detail, fmt.Errorf("%s", detail)
	}
	if fout, ferr := RunNetwork(root, "fetch", "-q", "origin", "--", branch); ferr != nil {
		detail := fmt.Sprintf("push rejected (non-fast-forward); fetch origin %s failed: %s — original: %s", branch, fout, out)
		return detail, fmt.Errorf("%s", detail)
	}
	if rout, rerr := Run(root, "rebase", "origin/"+branch); rerr != nil {
		_, _ = Run(root, "rebase", "--abort")
		detail := fmt.Sprintf("push rejected (non-fast-forward); rebase onto origin/%s failed and was aborted: %s — original push error: %s", branch, rout, out)
		return detail, fmt.Errorf("%s", detail)
	}
	return Push(root, branch)
}

// IsAncestor reports whether commit is an ancestor of (already merged into)
// ref — the trunk-membership question a "did this land?" check ultimately
// needs answered. Callers that care about GitHub's current state, not a
// possibly-stale local clone, should fetch first (e.g. RunNetwork(dir,
// "fetch", "origin") then compare against "origin/main") rather than trust
// whatever a prior checkout happened to have on disk: comparing a branch
// against a local main that hasn't been fetched since is exactly the false
// positive that mistook an in-flight `gh pr merge --auto` for an orphaned
// branch (tasks 157, 160 — the PR was landing, not abandoned).
func IsAncestor(dir, commit, ref string) (bool, error) {
	_, err := Run(dir, "merge-base", "--is-ancestor", commit, ref)
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// TipOnFirstParentMainline reports whether commit's tip sits on ref's
// first-parent mainline — i.e. commit IS trunk itself, not a side branch that
// entered trunk through a merge.
//
// This exists to tell a dead spawn apart from landed work when both are
// ancestors of trunk. A branch with no commits of its own is trivially an
// ancestor of trunk (dacli 168, 241), so IsAncestor alone would call a spawn
// that died before committing "merged" and force-accept an empty branch as a
// done task. A raw `rev-list --count trunk..branch == 0` guard cannot separate
// the two: a branch merged locally is ALSO zero commits ahead. The reliable
// signal is topology — dacli's local integrate is always a --no-ff merge
// (Merge above), so a branch that really landed enters trunk as a merge
// commit's SECOND parent and is NOT on the first-parent line, whereas a dead
// spawn's tip is a mainline trunk commit and IS.
func TipOnFirstParentMainline(dir, commit, ref string) (bool, error) {
	tip, err := Run(dir, "rev-parse", "--verify", commit+"^{commit}")
	if err != nil {
		return false, err
	}
	tip = strings.TrimSpace(tip)
	out, err := Run(dir, "rev-list", "--first-parent", ref)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == tip {
			return true, nil
		}
	}
	return false, nil
}

// isNonFastForward reports whether git push output names the "remote has
// commits you don't have" rejection specifically — as opposed to some other
// push failure (auth, protected branch, network) that PushSync must not mask
// behind a pointless rebase attempt.
func isNonFastForward(s string) bool {
	s = strings.ToLower(s)
	for _, sig := range []string{"non-fast-forward", "fetch first", "[rejected]", "behind its remote"} {
		if strings.Contains(s, sig) {
			return true
		}
	}
	return false
}
