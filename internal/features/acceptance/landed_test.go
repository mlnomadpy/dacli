package acceptance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func stubLandingGH(t *testing.T, state string) {
	t.Helper()
	old := runLandingGH
	runLandingGH = func(string, ...string) (string, error) {
		if state == "" {
			return "[]", nil
		}
		return fmt.Sprintf(`[{"state":%q}]`, state), nil
	}
	t.Cleanup(func() { runLandingGH = old })
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// landedFixture is a real repo with a trunk and one task, so the check runs
// against git rather than a stub — the whole point is that it answers a
// question about actual history.
func landedFixture(t *testing.T) (*workspace.Workspace, *store.Task) {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", dir)
	git(t, dir, "-C", dir, "config", "user.email", "x@x")
	git(t, dir, "-C", dir, "config", "user.name", "x")
	git(t, dir, "-C", dir, "checkout", "-q", "-b", "main")

	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "core", "ship the thing", store.TaskOpts{Accept: []string{"it works"}})
	if err != nil {
		t.Fatal(err)
	}
	git(t, dir, "-C", dir, "add", "-A")
	git(t, dir, "-C", dir, "commit", "-q", "-m", "base")
	return w, task
}

// The reported failure, reproduced: the task's branch has commits that never
// reached trunk. A build-and-test verify passes regardless — that is exactly
// how issue #382's run reported done:15/21 while the commands did not exist.
func TestUnlandedBranchIsDetected(t *testing.T) {
	w, task := landedFixture(t)
	branch := store.TaskBranch(task)

	git(t, w.Root, "-C", w.Root, "checkout", "-q", "-b", branch)
	git(t, w.Root, "-C", w.Root, "commit", "-q", "--allow-empty", "-m", "the deliverable")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "main")

	st, got := checkLanded(w, task, "main")
	if st != landingUnlanded {
		t.Errorf("state = %v, want landingUnlanded — the branch is not in trunk", st)
	}
	if got != branch {
		t.Errorf("branch = %q, want %q", got, branch)
	}
	if ev := landingEvidence(st, got, "main"); !strings.Contains(ev, "NOT in main") {
		t.Errorf("evidence must say the deliverable is missing, got %q", ev)
	}
}

// Once merged, the same task reads as landed — the check must not cry wolf on
// work that actually shipped, or operators will learn to ignore it.
func TestMergedBranchReadsAsLanded(t *testing.T) {
	w, task := landedFixture(t)
	branch := store.TaskBranch(task)

	git(t, w.Root, "-C", w.Root, "checkout", "-q", "-b", branch)
	git(t, w.Root, "-C", w.Root, "commit", "-q", "--allow-empty", "-m", "the deliverable")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "main")
	git(t, w.Root, "-C", w.Root, "merge", "-q", "--no-ff", "-m", "merge", branch)

	if st, _ := checkLanded(w, task, "main"); st != landingLanded {
		t.Errorf("state = %v, want landingLanded after the merge", st)
	}
}

// PRs 509, 510, and 511 had this topology: GitHub squash-merged each PR,
// placing a new commit on main while the original task commit remained outside
// main. accept must trust the PR's MERGED state, exactly as `dacli pr status`
// does, so the normal strict path succeeds without --allow-unlanded.
func TestSquashMergedPRReadsAsLandedWithoutOverride(t *testing.T) {
	w, task := landedFixture(t)
	branch := store.TaskBranch(task)
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "-b", branch)
	git(t, w.Root, "-C", w.Root, "commit", "-q", "--allow-empty", "-m", "original task commit")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "main")
	git(t, w.Root, "-C", w.Root, "commit", "-q", "--allow-empty", "-m", "GitHub squash commit for PRs 509 510 511")
	stubLandingGH(t, "MERGED")

	if st, _ := checkLanded(w, task, "main"); st != landingLanded {
		t.Fatalf("confirmed squash-merged PR = %v, want landed without --allow-unlanded", st)
	}
}

func TestClosedUnmergedPRWithSimilarDiffRemainsUnlanded(t *testing.T) {
	w, task := landedFixture(t)
	branch := store.TaskBranch(task)
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "-b", branch)
	if err := os.WriteFile(filepath.Join(w.Root, "deliverable.txt"), []byte("same diff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, w.Root, "-C", w.Root, "add", "deliverable.txt")
	git(t, w.Root, "-C", w.Root, "commit", "-q", "-m", "task implementation")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "main")
	if err := os.WriteFile(filepath.Join(w.Root, "deliverable.txt"), []byte("same diff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, w.Root, "-C", w.Root, "add", "deliverable.txt")
	git(t, w.Root, "-C", w.Root, "commit", "-q", "-m", "unrelated implementation")
	stubLandingGH(t, "CLOSED")

	if st, _ := checkLanded(w, task, "main"); st != landingUnlanded {
		t.Fatalf("closed-unmerged PR with similar diff = %v, want unlanded", st)
	}
}

// No branch is legitimate — work committed straight to trunk, a docs task, a
// record task — and must not be reported as a missing deliverable.
func TestNoBranchIsNotAFailure(t *testing.T) {
	w, task := landedFixture(t)
	st, _ := checkLanded(w, task, "main")
	if st != landingNoBranch {
		t.Errorf("state = %v, want landingNoBranch — a task with no branch has nothing to contradict", st)
	}
	if ev := landingEvidence(st, store.TaskBranch(task), "main"); strings.Contains(ev, "NOT in main") {
		t.Errorf("a branchless task must not read as an unlanded deliverable: %q", ev)
	}
}

// The strict refusal is a policy answer (exit 3) and names the way out, so a
// supervisor never retries it unchanged.
func TestUnlandedRefusalNamesTheWayOut(t *testing.T) {
	err := unlandedRefusal(7, "dacli/007-x", "main")
	if got := clikit.ExitCode(err); got != 3 {
		t.Errorf("exit %d, want 3 (policy refusal)", got)
	}
	if !strings.Contains(err.Error(), "--allow-unlanded") {
		t.Errorf("refusal must name the flag that overrides it: %v", err)
	}
}

// An agent's "I am finished" must be answerable whichever verb produced it.
//
// There are two proposal channels and an agent cannot be expected to know the
// difference: `dacli accept` files an accept-propose COMMENT, while `dacli
// task done` — which is what the protocol preamble tells agents to run —
// files a propose-status EVENT. Only the comment channel was consumed, so a
// `task done` proposal was invisible to `accept --all`.
//
// That was one leg of a three-way deadlock (task 312): the agent's claim is
// not applied until sync, so it cannot check its own acceptance boxes; sync
// then refuses its propose:done because those boxes are unmet; and accept
// could not see the request. Agents committed real work and the loop halted
// on "no net progress" with the work finished and unlandable.
func TestBothProposalChannelsCountAsACloseRequest(t *testing.T) {
	comment := &eventlog.Event{Kind: model.EventComment, Body: proposePrefix + " done"}
	if !isCloseRequest(comment) {
		t.Error("an accept-propose comment must read as a close request")
	}

	proposeDone := &eventlog.Event{Kind: model.EventProposeStatus, Body: "propose: done"}
	if !isCloseRequest(proposeDone) {
		t.Error("a propose-status done must read as a close request — it is what `task done` files, and what agents are told to run")
	}

	// Other status proposals are NOT close requests: blocked means the agent
	// wants attention, not a close.
	blocked := &eventlog.Event{Kind: model.EventProposeStatus, Body: "propose: blocked"}
	if isCloseRequest(blocked) {
		t.Error("propose: blocked must not be treated as a request to close")
	}
	// And an ordinary comment is just a comment.
	chat := &eventlog.Event{Kind: model.EventComment, Body: "looks fine to me"}
	if isCloseRequest(chat) {
		t.Error("a plain comment must not be treated as a close request")
	}
}

// A sprint lands a batch on its own branch and takes ONE pull request to main
// at the end. During that window the work is not in main and is not supposed to
// be, so the landing check asked the wrong question: every accept of a sprint
// warned that landed work was "NOT in trunk". A warning that is wrong on every
// run is one nobody reads when it is right (dacli 342, hit three times in the
// sprint that produced this fix).
func TestLandingChecksTheBranchTheWorkIsIntegratedInto(t *testing.T) {
	w, task := landedFixture(t)
	branch := store.TaskBranch(task)

	// The task's work lands on sprint/1, not on main.
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "-b", "sprint/1")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "-b", branch)
	git(t, w.Root, "-C", w.Root, "commit", "-q", "--allow-empty", "-m", "the deliverable")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "sprint/1")
	git(t, w.Root, "-C", w.Root, "merge", "-q", "--no-ff", branch, "-m", "merge")
	git(t, w.Root, "-C", w.Root, "checkout", "-q", "main")

	// Against the branch it actually landed on: landed, and the record NAMES
	// that branch rather than calling it "trunk", which would be false.
	if st, _ := checkLanded(w, task, landingTarget(w, "sprint/1")); st != landingLanded {
		t.Errorf("state = %v against sprint/1, want landingLanded — it merged there", st)
	}
	ev := landingEvidence(landingLanded, branch, landingTarget(w, "sprint/1"))
	if !strings.Contains(ev, "sprint/1") {
		t.Errorf("the record must name where the work landed, got %q", ev)
	}

	// And the check must NOT become a rubber stamp: against main, where the
	// work genuinely is not, it still says so.
	if st, _ := checkLanded(w, task, landingTarget(w, "")); st != landingUnlanded {
		t.Errorf("state = %v against the resolved trunk, want landingUnlanded — it is not in main", st)
	}
}

// landingTarget falls back to the repository's trunk when no --into is given,
// so a caller that names nothing behaves exactly as before.
func TestLandingTargetDefaultsToTrunk(t *testing.T) {
	w, _ := landedFixture(t)
	if got, want := landingTarget(w, ""), trunkBranch(w); got != want {
		t.Errorf("landingTarget(\"\") = %q, want the resolved trunk %q", got, want)
	}
	if got := landingTarget(w, "sprint/9"); got != "sprint/9" {
		t.Errorf("an explicit target must win, got %q", got)
	}
}
