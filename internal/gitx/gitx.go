// Package gitx is the shared git-operations layer — the entity-level plumbing
// that the vcs and execution slices both build on (slices never import each
// other, so shared git logic lives here). It exists for the parallel-agent
// lifecycle: isolated worktrees, push/PR, and conflict-aware merges.
package gitx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/commandresult"
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

// ErrLeaseRequired marks a task-branch divergence that policy will not merge
// automatically. Callers expose it as exit 3: retrying cannot make an
// ambiguous history replacement safe.
var ErrLeaseRequired = errors.New("lease-protected push requires operator decision")

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
	operation := "git"
	if len(args) > 0 {
		// One verb identifies the operation without copying argv, which may carry
		// credentials or private remote URLs into an error (issue #876).
		operation += " " + args[0]
	}
	out, err := commandresult.Run(cmd, commandresult.RunOptions{
		Operation:     operation,
		WorkspaceRoot: dir,
		TimedOut: func() bool {
			return ctx.Err() == context.DeadlineExceeded
		},
	})
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
//
// When it REUSES an existing branch it fast-forwards that branch to trunk
// first, where doing so is provably safe. A new branch is cut from HEAD and so
// starts current; an existing one was left wherever its last run ended, and
// nothing advanced it. For a RECURRING task — the standing continuous-improvement
// anchor is the one that matters — the branch persists across every cycle, so
// an auditor was handed a tree arbitrarily far behind trunk and asked to find
// bugs in code where the fixes had not been applied. Its findings then
// re-report defects that are already closed, and those phantom findings spawn
// fixer agents that rebuild work that already exists (issue #441).
//
// What makes it safe is `merge --ff-only`, and only that: it succeeds exactly
// when the branch is an ancestor of trunk — carries no commit of its own — and
// refuses otherwise. So a branch with real unmerged work comes through
// untouched and is reported via freshened=false. Rewriting or auto-merging an
// agent's work at spawn time would be a far worse bug than the staleness this
// fixes, and the safety of that must not rest on a check we could forget: it
// rests on git's.
//
// The explicit IsAncestor call below is therefore a PRE-FILTER, not the guard —
// it skips the merge and two rev-parses for the common already-diverged case.
// Deleting it changes performance, not behaviour. Stated plainly because a
// comment that credits the wrong line for a guarantee is how the next reader
// removes the line that actually provides it.
//
// trunk may be empty (a repo with no trunk, most unit tests), in which case
// this is the old behaviour verbatim.
func AddWorktree(root, path, branch, trunk string) (freshened bool, err error) {
	if !BranchExists(root, branch) {
		if _, err := Run(root, "worktree", "add", "-b", branch, path); err != nil {
			return false, fmt.Errorf("worktree add -b: %w", err)
		}
		return false, nil // cut from HEAD: current by construction
	}
	if _, err := Run(root, "worktree", "add", path, branch); err != nil {
		return false, fmt.Errorf("worktree add: %w", err)
	}
	return freshenToTrunk(root, path, branch, trunk), nil
}

// freshenToTrunk fast-forwards an existing branch checked out at path up to
// trunk, and reports whether it moved. Best-effort throughout: every failure
// path leaves the worktree usable at its old tip, because a stale tree is a
// degraded run while a failed spawn is no run at all.
func freshenToTrunk(root, path, branch, trunk string) bool {
	if trunk == "" || branch == trunk {
		return false
	}
	// Prefer the local trunk ref, then origin's — the same both-refs rule the
	// landing check settled on, for the same reason: either can be the stale
	// one, so trusting a fixed order silently freshens to an old commit.
	for _, ref := range []string{"refs/heads/" + trunk, "refs/remotes/origin/" + trunk} {
		if _, err := Run(root, "rev-parse", "--verify", "--quiet", ref); err != nil {
			continue
		}
		// Pre-filter only — `merge --ff-only` below enforces this itself and is
		// what makes the operation safe. This just avoids invoking it, and the
		// two rev-parses around it, for a branch that has obviously diverged.
		if ancestor, err := IsAncestor(root, branch, ref); err != nil || !ancestor {
			continue
		}
		// Report movement, not the command's exit status. A branch already AT
		// trunk's tip is trivially an ancestor and `merge --ff-only` succeeds
		// as a no-op there, so returning true on success alone would announce a
		// freshening that never happened — a guard reporting work it did not do.
		before := revParse(path, "HEAD")
		if _, err := Run(path, "merge", "--ff-only", ref); err != nil {
			continue // git refused: not worth failing a spawn over
		}
		after := revParse(path, "HEAD")
		return before != "" && after != "" && before != after
	}
	return false
}

func revParse(dir, ref string) string {
	out, err := Run(dir, "rev-parse", ref)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
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
	registered, registeredErr := ListWorktrees(root)
	if _, removeErr := Run(root, "worktree", "remove", "--force", path); removeErr != nil {
		// An interrupted teardown can leave the directory without its .git link.
		// Git still lists that checkout as "prunable", but `worktree remove`
		// refuses it. Prune the broken registration, then remove the orphaned
		// directory only after Git confirms it no longer owns the path (dacli 439).
		if registeredErr != nil || !worktreeRegistered(registered, path) {
			return fmt.Errorf("worktree remove: %w", removeErr)
		}
		_, _ = Run(root, "worktree", "prune", "--expire", "now")
		wts, listErr := ListWorktrees(root)
		if listErr != nil || worktreeRegistered(wts, path) {
			return fmt.Errorf("worktree remove: %w", removeErr)
		}
		cleanRoot, cleanPath := filepath.Clean(root), filepath.Clean(path)
		if cleanPath == "." || cleanPath == cleanRoot {
			return fmt.Errorf("worktree remove: refusing orphan cleanup at repository root %s", cleanRoot)
		}
		if err := os.RemoveAll(cleanPath); err != nil {
			return fmt.Errorf("worktree remove orphan: %w", err)
		}
	}
	return nil
}

// RemoveCleanWorktree performs the narrow, non-forcing removal used by the
// repository cleanup planner. Unlike RemoveWorktree it has no interrupted-run
// fallback that removes an orphan directory: a cleanup plan may execute only
// the exact recoverable git operation it displayed, or fail closed.
func RemoveCleanWorktree(root, path string) error {
	_, err := Run(root, "worktree", "remove", "--", path)
	return err
}

func worktreeRegistered(wts []Worktree, path string) bool {
	want := filepath.Clean(path)
	for _, wt := range wts {
		if filepath.Clean(wt.Path) == want {
			return true
		}
	}
	return false
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
	if _, mergeErr := Run(root, "merge", "--no-ff", "-m", message, branch); mergeErr != nil {
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
			return nil, fmt.Errorf("git merge %s failed: %w", branch, mergeErr)
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
		return out, fmt.Errorf("fetch origin %s: %w", branch, err)
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
		if err == nil {
			return "ordinary fast-forward push: " + out, nil
		}
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
		return detail, fmt.Errorf("%s: %w", detail, ferr)
	}
	// A task branch rebased onto current trunk must never be reconciled by
	// rebasing onto its old remote tip (GitHub #726): that makes obsolete remote
	// commits ancestors again and can enlarge the PR while reporting success.
	// When every remote-only patch is already represented on trunk, replacing
	// the stale ref is lossless and an exact lease makes the observed OID part of
	// the write. Any other divergence is ambiguous and must be an exit-3 refusal.
	if strings.HasPrefix(branch, "dacli/") {
		return pushRebasedTaskWithLease(root, branch, out)
	}
	if rout, rerr := Run(root, "rebase", "origin/"+branch); rerr != nil {
		_, _ = Run(root, "rebase", "--abort")
		detail := fmt.Sprintf("push rejected (non-fast-forward); rebase onto origin/%s failed and was aborted: %s — original push error: %s", branch, rout, out)
		return detail, fmt.Errorf("%s: %w", detail, rerr)
	}
	retry, retryErr := Push(root, branch)
	if retryErr == nil {
		return "synchronized push after rebase: " + retry, nil
	}
	return retry, retryErr
}

func pushRebasedTaskWithLease(root, branch, original string) (string, error) {
	if out, err := RunNetwork(root, "fetch", "-q", "origin", "--", "main"); err != nil {
		detail := fmt.Sprintf("push rejected (non-fast-forward); fetch origin main failed: %s — original: %s", out, original)
		return detail, fmt.Errorf("%s: %w", detail, err)
	}
	remoteRef := "refs/remotes/origin/" + branch
	remoteOID, err := Run(root, "rev-parse", "--verify", remoteRef+"^{commit}")
	if err != nil {
		return original, err
	}
	landingOID, err := Run(root, "rev-parse", "--verify", "refs/remotes/origin/main^{commit}")
	if err != nil {
		return original, err
	}
	base, err := Run(root, "merge-base", branch, "origin/main")
	if err != nil {
		return original, err
	}
	cherry, cherryErr := Run(root, "cherry", "origin/main", "origin/"+branch)
	patchesLanded := cherryErr == nil
	for _, line := range strings.Split(cherry, "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "-") {
			patchesLanded = false
		}
	}
	recovery := fmt.Sprintf("git push --force-with-lease=refs/heads/%s:%s origin %s", branch, strings.TrimSpace(remoteOID), branch)
	if strings.TrimSpace(base) != strings.TrimSpace(landingOID) || !patchesLanded {
		detail := fmt.Sprintf("push refused: local %s and origin/%s have ambiguous divergent history; local history was not changed. Inspect the fetched remote, then if replacement is intended run exactly: %s", branch, branch, recovery)
		return detail, fmt.Errorf("%w: %s", ErrLeaseRequired, detail)
	}
	lease := fmt.Sprintf("--force-with-lease=refs/heads/%s:%s", branch, strings.TrimSpace(remoteOID))
	leaseOut, leaseErr := RunNetwork(root, "push", "-u", lease, "origin", "--", branch)
	if leaseErr != nil {
		return leaseOut, leaseErr
	}
	return "lease-protected history rewrite: " + leaseOut, nil
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
