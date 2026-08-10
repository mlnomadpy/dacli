package acceptance

import (
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// landingState aliases store.LandingState: the check itself lives in the
// entity layer (internal/store/landing.go) so `ship`, which cannot import
// this feature slice, can run the exact same truthful check to CORRECT a
// landing verdict accept had to record before ship's own integrate step had
// run (dacli 329). accept keeps these lower-case local names so its own
// call sites and tests read unchanged.
type landingState = store.LandingState

const (
	landingUnknown  = store.LandingUnknown
	landingNoBranch = store.LandingNoBranch
	landingLanded   = store.LandingLanded
	landingUnlanded = store.LandingUnlanded
)

func checkLanded(w *workspace.Workspace, t *store.Task, trunk string) (landingState, string) {
	return store.CheckLanded(w, t, trunk)
}

func landingEvidence(st landingState, branch, target string) string {
	return store.LandingEvidence(st, branch, target)
}

func trunkBranch(w *workspace.Workspace) string {
	return store.TrunkBranch(w)
}

// unlandedRefusal is the strict-mode answer. It is a policy refusal (exit 3):
// retrying it unchanged cannot succeed, and the way forward — merge it, or say
// out loud that you are closing it anyway — is named.
func unlandedRefusal(seq int, branch, trunk string) error {
	return clikit.Refusedf(
		"task %03d has commits on %s that are NOT in %s — closing it now would record work the trunk never received (the failure issue #382 reported: done:15/21 while the commands did not exist). Merge the branch, or pass --allow-unlanded to close it deliberately",
		seq, branch, trunk)
}

// landingTarget is the branch a close should be checked against: the explicit
// --into when the caller named one, otherwise the repository's trunk.
//
// A sprint lands a batch on its own branch and takes one pull request to main
// at the end, so during that window "in trunk" is the wrong question — the work
// belongs on sprint/N, not yet on main. Without this the check warned on every
// accept of a sprint, and a warning that is wrong every time is one nobody
// reads when it is right (dacli 342).
func landingTarget(w *workspace.Workspace, into string) string {
	if into != "" {
		return into
	}
	return trunkBranch(w)
}
