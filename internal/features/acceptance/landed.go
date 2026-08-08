package acceptance

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// landingState is what we could learn about whether a task's work reached
// trunk. It is deliberately three-valued: "no branch" and "cannot tell" are
// different from "landed", and neither should be reported as success.
type landingState int

const (
	landingUnknown  landingState = iota // no git, no trunk, or the check failed
	landingNoBranch                     // the task never had a branch
	landingLanded                       // the branch is an ancestor of trunk
	landingUnlanded                     // the branch exists and is NOT in trunk
)

// checkLanded asks whether the task's branch has reached trunk.
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
func checkLanded(w *workspace.Workspace, t *store.Task, trunk string) (landingState, string) {
	if !gitx.Available() || trunk == "" {
		return landingUnknown, ""
	}
	branch := store.TaskBranch(t)

	// Prefer the remote ref when there is one: a local branch can be stale or
	// absent on a machine that never checked it out, while origin/<branch> is
	// what a PR actually merges.
	ref := ""
	for _, cand := range []string{"refs/remotes/origin/" + branch, "refs/heads/" + branch} {
		if _, err := gitx.Run(w.Root, "rev-parse", "--verify", "--quiet", cand); err == nil {
			ref = cand
			break
		}
	}
	if ref == "" {
		// No branch at all. Common and legitimate: work committed straight to
		// trunk, a docs-only task, or a record task. Nothing to contradict.
		return landingNoBranch, branch
	}

	for _, trunkRef := range []string{"refs/remotes/origin/" + trunk, "refs/heads/" + trunk} {
		if _, err := gitx.Run(w.Root, "rev-parse", "--verify", "--quiet", trunkRef); err != nil {
			continue
		}
		in, err := gitx.IsAncestor(w.Root, ref, trunkRef)
		if err != nil {
			return landingUnknown, branch
		}
		if in {
			return landingLanded, branch
		}
		return landingUnlanded, branch
	}
	return landingUnknown, branch
}

// landingEvidence renders the one Log line recorded on the task, so the
// trajectory states what was known about the deliverable at close time rather
// than implying more than was checked.
func landingEvidence(st landingState, branch string) string {
	switch st {
	case landingLanded:
		return fmt.Sprintf("deliverable: %s is merged into trunk", branch)
	case landingUnlanded:
		return fmt.Sprintf("deliverable: %s exists but is NOT in trunk — closed anyway", branch)
	case landingNoBranch:
		return fmt.Sprintf("deliverable: no %s branch — nothing to check against trunk", branch)
	default:
		return "deliverable: could not be checked against trunk"
	}
}

// unlandedRefusal is the strict-mode answer. It is a policy refusal (exit 3):
// retrying it unchanged cannot succeed, and the way forward — merge it, or say
// out loud that you are closing it anyway — is named.
func unlandedRefusal(seq int, branch, trunk string) error {
	return clikit.Refusedf(
		"task %03d has commits on %s that are NOT in %s — closing it now would record work the trunk never received (the failure issue #382 reported: done:15/21 while the commands did not exist). Merge the branch, or pass --allow-unlanded to close it deliberately",
		seq, branch, trunk)
}

// trunkBranch resolves the branch a task's work is expected to land on. It
// prefers the repo's actual HEAD-of-origin, falling back to the conventional
// names, so a repo using `master` or a renamed default is not silently
// reported as "cannot tell".
func trunkBranch(w *workspace.Workspace) string {
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
