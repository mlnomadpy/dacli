package acceptance

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

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
	if ev := landingEvidence(st, got); !strings.Contains(ev, "NOT in trunk") {
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

// No branch is legitimate — work committed straight to trunk, a docs task, a
// record task — and must not be reported as a missing deliverable.
func TestNoBranchIsNotAFailure(t *testing.T) {
	w, task := landedFixture(t)
	st, _ := checkLanded(w, task, "main")
	if st != landingNoBranch {
		t.Errorf("state = %v, want landingNoBranch — a task with no branch has nothing to contradict", st)
	}
	if ev := landingEvidence(st, store.TaskBranch(task)); strings.Contains(ev, "NOT in trunk") {
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
