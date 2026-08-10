package store

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// LandingState is what we could learn about whether a task's work reached
// trunk. It is deliberately three-valued: "no branch" and "cannot tell" are
// different from "landed", and neither should be reported as success.
//
// This lives in the entity layer (not a feature slice) because both
// `accept` (features/acceptance) and `ship` (features/ship) need the exact
// same truthful check — accept to certify a close, ship to CORRECT that
// certification once its own later integrate step actually lands the
// branch accept could not yet see (dacli 329).
type LandingState int

const (
	LandingUnknown  LandingState = iota // no git, no trunk, or the check failed
	LandingNoBranch                     // the task never had a branch
	LandingLanded                       // the branch is an ancestor of trunk
	LandingUnlanded                     // the branch exists and is NOT in trunk
)

// CheckLanded asks whether the task's branch has reached trunk.
//
// This is the gap issue #382 called its most serious finding: `accept --verify
// "go build && go test"` passes whether or not the task's deliverable landed.
// On that reporter's run, PR merges had failed (DIRTY), the code never merged,
// and accept still marked the tasks done — the CLI reported done:15/21 while
// the vector-search commands it claimed to have built did not exist at all.
// A build passing proves the TREE compiles, not that this task's work is in it.
//
// The check is intentionally cheap and local: does a branch named for this
// task exist, and is it an ancestor of trunk? That answers the exact question
// a coarse verify cannot.
func CheckLanded(w *workspace.Workspace, t *Task, trunk string) (LandingState, string) {
	if !gitx.Available() || trunk == "" {
		return LandingUnknown, ""
	}
	branch, sha, found := ResolveBranchRef(w, t)
	if !found {
		// No branch at all. Common and legitimate: work committed straight to
		// trunk, a docs-only task, or a record task. Nothing to contradict.
		return LandingNoBranch, branch
	}
	return LandingOfRef(w, sha, trunk), branch
}

// ResolveBranchRef finds the commit a task's branch currently points to,
// preferring the remote ref (a local branch can be stale or absent on a
// machine that never checked it out, while origin/<branch> is what a PR
// actually merges) — exactly the ref CheckLanded itself would resolve.
//
// Exposed separately so a caller can snapshot it BEFORE an operation that
// might delete the branch (a clean local merge does exactly that once it
// lands — see vcs.mergeTask), and still answer "was THIS commit merged?"
// afterward via LandingOfRef, when the branch name alone can no longer say.
func ResolveBranchRef(w *workspace.Workspace, t *Task) (branch, sha string, found bool) {
	branch = TaskBranch(t)
	if !gitx.Available() {
		return branch, "", false
	}
	for _, cand := range []string{"refs/remotes/origin/" + branch, "refs/heads/" + branch} {
		if out, err := gitx.Run(w.Root, "rev-parse", "--verify", "--quiet", cand); err == nil {
			return branch, strings.TrimSpace(out), true
		}
	}
	return branch, "", false
}

// LandingOfRef answers whether an already-resolved commit (sha) has reached
// trunk. Separated from CheckLanded's branch lookup so a caller holding a
// commit captured before the branch was deleted can still ask the question.
func LandingOfRef(w *workspace.Workspace, sha, trunk string) LandingState {
	if !gitx.Available() || trunk == "" || sha == "" {
		return LandingUnknown
	}
	for _, trunkRef := range []string{"refs/remotes/origin/" + trunk, "refs/heads/" + trunk} {
		if _, err := gitx.Run(w.Root, "rev-parse", "--verify", "--quiet", trunkRef); err != nil {
			continue
		}
		in, err := gitx.IsAncestor(w.Root, sha, trunkRef)
		if err != nil {
			return LandingUnknown
		}
		if in {
			return LandingLanded
		}
		return LandingUnlanded
	}
	return LandingUnknown
}

// LandingEvidence renders the one Log line recorded on the task, so the
// trajectory states what was known about the deliverable at close time rather
// than implying more than was checked.
// The TARGET is named, never called "trunk" generically. During a sprint the
// work lands on sprint/N and takes one pull request to main at the end, so a
// record saying "merged into trunk" would be false about where it actually
// went — and the record is the product (dacli 342).
func LandingEvidence(st LandingState, branch, target string) string {
	if target == "" {
		target = "trunk"
	}
	switch st {
	case LandingLanded:
		return fmt.Sprintf("deliverable: %s is merged into %s", branch, target)
	case LandingUnlanded:
		return fmt.Sprintf("deliverable: %s exists but is NOT in %s — closed anyway", branch, target)
	case LandingNoBranch:
		return fmt.Sprintf("deliverable: no %s branch — nothing to check against %s", branch, target)
	default:
		return fmt.Sprintf("deliverable: could not be checked against %s", target)
	}
}

// TrunkBranch resolves the branch a task's work is expected to land on. It
// prefers the repo's actual HEAD-of-origin, falling back to the conventional
// names, so a repo using `master` or a renamed default is not silently
// reported as "cannot tell".
func TrunkBranch(w *workspace.Workspace) string {
	if out, err := gitx.Run(w.Root, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if s := strings.TrimSpace(out); s != "" {
			return strings.TrimPrefix(s, "origin/")
		}
	}
	for _, cand := range []string{"main", "master"} {
		if gitx.BranchExists(w.Root, cand) {
			return cand
		}
	}
	return ""
}
