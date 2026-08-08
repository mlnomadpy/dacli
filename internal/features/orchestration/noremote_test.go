package orchestration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func noRemoteRepo(t *testing.T) *workspace.Workspace {
	t.Helper()
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "-Af", "--", ".dacli")
	gitAt(t, dir, "commit", "-q", "-m", "base")
	return w
}

// With no `origin`, the PR path cannot land anything: gh fails on every call,
// every branch reads "unknown", no accept resolves, and the loop re-picks the
// same tasks forever. That is issue #382's first symptom, and the prevention
// is to not choose the PR path at all when there is no remote.
func TestLoopLandsLocallyWhenThereIsNoRemote(t *testing.T) {
	w := noRemoteRepo(t)
	f, err := clikit.ParseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	if prMode(w, f) {
		t.Error("the loop chose the PR path with no origin — every landing check would dead-end and no task would ever close")
	}

	// An explicit --no-pr is unchanged.
	off, _ := clikit.ParseFlags([]string{"--no-pr"})
	if prMode(w, off) {
		t.Error("--no-pr must always disable the PR path")
	}
}

// A merge into the LOCAL trunk is a real landing. Reporting it as "unknown"
// is what left tasks open forever in a no-remote workspace.
func TestLandStatusSeesALocalMerge(t *testing.T) {
	w := noRemoteRepo(t)
	d := &driver{w: w, trunkBranch: "main", cfg: loopCfg{project: "core"}}

	const branch = "dacli/001-a-task"
	gitAt(t, w.Root, "checkout", "-q", "-b", branch)
	gitAt(t, w.Root, "commit", "-q", "--allow-empty", "-m", "the work")
	gitAt(t, w.Root, "checkout", "-q", "main")

	if got := d.prLandStatus(branch); got != "orphaned" {
		t.Errorf("before the merge: got %q, want orphaned (work exists, not in trunk)", got)
	}

	gitAt(t, w.Root, "merge", "-q", "--no-ff", "-m", "merge", branch)
	if got := d.prLandStatus(branch); got != "merged" {
		t.Errorf("after a LOCAL merge: got %q, want merged — a local landing is a landing, and calling it unknown is what made the loop thrash", got)
	}
}

// A branch that never committed must not read as merged just because it is
// trivially an ancestor of trunk — that is the path that force-accepts empty
// work as done (dacli 168), and it must keep holding with no remote.
func TestLandStatusStillRefusesAnEmptyBranchLocally(t *testing.T) {
	w := noRemoteRepo(t)
	d := &driver{w: w, trunkBranch: "main", cfg: loopCfg{project: "core"}}

	const branch = "dacli/002-empty"
	gitAt(t, w.Root, "branch", branch)
	if got := d.prLandStatus(branch); got != "orphaned" {
		t.Errorf("an empty branch got %q, want orphaned — a spawn that never committed is not a landing", got)
	}
}

// hasOrigin is the whole distinction between "the PR has not merged yet" and
// "there is no GitHub in this picture" — two states that used to collapse into
// the one value that never resolves.
func TestHasOriginDistinguishesNoRemote(t *testing.T) {
	w := noRemoteRepo(t)
	d := &driver{w: w}
	if d.hasOrigin() {
		t.Error("a repo with no remotes reported an origin")
	}
	gitAt(t, w.Root, "remote", "add", "origin", "https://example.invalid/x.git")
	if !d.hasOrigin() {
		t.Error("an origin was added but not detected")
	}
}
