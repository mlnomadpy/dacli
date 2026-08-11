// The default `ship` path merges each task branch into trunk LOCALLY, records
// the landing verdict, and only then pushes. So at the moment the verdict is
// written, `origin/main` is one commit behind `refs/heads/main` by
// construction — every single time.
//
// LandingOfRef consulted both refs, which reads as thorough, but returned its
// verdict from the FIRST ref that existed rather than the first that answered
// "landed". With an origin configured, the local ref holding the merge was
// unreachable code, and ship stamped a permanent "NOT in main — closed anyway"
// onto work it had just merged into main. That is the record disagreeing with
// reality, on the default path, committed to the trajectory that is the
// product.
package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// runOutErr is runOut's non-fatal sibling: these tests assert on git commands
// that are EXPECTED to fail (`merge-base --is-ancestor` exits 1 for "no"), so
// the error is the answer rather than a test failure.
func runOutErr(dir string, args ...string) (string, error) {
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := c.CombinedOutput()
	return string(out), err
}

// staleOriginRepo builds the exact shape ship is in at step 2b: a task branch
// merged into local main, an origin whose main still points at the base commit,
// and nothing pushed.
func staleOriginRepo(t *testing.T) (w *workspace.Workspace, merged string) {
	t.Helper()
	w = fiRepo(t)
	base := strings.TrimSpace(runOut(t, w.Root, "rev-parse", "HEAD"))

	// A real remote, not a fake ref: origin/main must exist and be genuinely
	// behind, which is what a pre-push working tree looks like.
	remote := t.TempDir()
	gitFI(t, remote, "init", "-q", "--bare")
	gitFI(t, w.Root, "remote", "add", "origin", remote)
	gitFI(t, w.Root, "push", "-q", "origin", "main")
	gitFI(t, w.Root, "fetch", "-q", "origin")

	// The task's work, merged into local main the way vcs.mergeTask does.
	gitFI(t, w.Root, "checkout", "-q", "-b", "task/001-work")
	if err := os.WriteFile(filepath.Join(w.Root, "b.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFI(t, w.Root, "add", "-A")
	gitFI(t, w.Root, "commit", "-q", "-m", "the deliverable")
	merged = strings.TrimSpace(runOut(t, w.Root, "rev-parse", "HEAD"))
	gitFI(t, w.Root, "checkout", "-q", "main")
	gitFI(t, w.Root, "merge", "-q", "--no-ff", "-m", "merge task/001-work", "task/001-work")

	// The premise, asserted rather than assumed: origin is stale and local main
	// holds the merge. If either stops being true the test below proves nothing.
	if got := strings.TrimSpace(runOut(t, w.Root, "rev-parse", "refs/remotes/origin/main")); got != base {
		t.Fatalf("setup: origin/main should still be at base %s, got %s", base, got)
	}
	if out, err := runOutErr(w.Root, "merge-base", "--is-ancestor", merged, "refs/heads/main"); err != nil {
		t.Fatalf("setup: the deliverable must be in local main: %v %s", err, out)
	}
	return w, merged
}

// TestLandingOfRefSeesLocalTrunkWhenOriginIsStale is the regression: the
// deliverable IS in main, and the verdict must say so.
func TestLandingOfRefSeesLocalTrunkWhenOriginIsStale(t *testing.T) {
	w, merged := staleOriginRepo(t)
	if got := LandingOfRef(w, merged, "main"); got != LandingLanded {
		t.Fatalf("a commit merged into local main must read as landed even while origin/main is behind, got %v", got)
	}
}

// TestLandingOfRefStillReportsGenuinelyUnlandedWork is the other half, and the
// one that stops the fix above from degenerating into "always landed": work
// that is in NEITHER ref must stay visibly unlanded. ship's error paths depend
// on this — a task blocked on a merge conflict is recorded from this same call.
func TestLandingOfRefStillReportsGenuinelyUnlandedWork(t *testing.T) {
	w, _ := staleOriginRepo(t)
	gitFI(t, w.Root, "checkout", "-q", "-b", "task/002-never-merged")
	if err := os.WriteFile(filepath.Join(w.Root, "c.txt"), []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFI(t, w.Root, "add", "-A")
	gitFI(t, w.Root, "commit", "-q", "-m", "work that never merged")
	orphan := strings.TrimSpace(runOut(t, w.Root, "rev-parse", "HEAD"))
	gitFI(t, w.Root, "checkout", "-q", "main")

	if got := LandingOfRef(w, orphan, "main"); got != LandingUnlanded {
		t.Fatalf("work in neither trunk ref must read as unlanded, got %v", got)
	}
}

// TestLandingOfRefSeesOriginWhenLocalTrunkIsStale is the mirror case, and the
// reason both refs are consulted rather than just one: on a machine that
// fetched but never fast-forwarded its local main, the merge lives only in
// origin/main. Whichever ref is behind, the answer is the same.
func TestLandingOfRefSeesOriginWhenLocalTrunkIsStale(t *testing.T) {
	w, merged := staleOriginRepo(t)
	// Push the merge, then rewind local main behind it — a fetched-not-merged
	// checkout.
	gitFI(t, w.Root, "push", "-q", "origin", "main")
	gitFI(t, w.Root, "fetch", "-q", "origin")
	gitFI(t, w.Root, "reset", "-q", "--hard", merged+"~1")

	if out, err := runOutErr(w.Root, "merge-base", "--is-ancestor", merged, "refs/heads/main"); err == nil {
		t.Fatalf("setup: local main should NOT contain the merge, got %s", out)
	}
	if got := LandingOfRef(w, merged, "main"); got != LandingLanded {
		t.Fatalf("a commit present in origin/main must read as landed, got %v", got)
	}
}

// --- the mirror fault, one level up: a false LANDED ---------------------
//
// Preferring one branch ref was wrong in the other direction too, and worse.
// A branch pushed once and then advanced locally has its deliverable ONLY in
// refs/heads/<branch>; ResolveBranchRef returned origin/<branch>. If that
// older commit had already merged — the ordinary case, since it is what got
// pushed and reviewed — the landing check found it in trunk and certified the
// task as landed while the deliverable sat in unmerged local commits.
//
// That is the exact failure issue #382 exists for: accept marking work done
// that trunk never received. A false unlanded is noise; a false landed is the
// record certifying something untrue.

// splitBranchRepo builds it: part one pushed and merged into trunk, part two
// committed locally and merged nowhere.
func splitBranchRepo(t *testing.T) (w *workspace.Workspace, task *Task, localTip string) {
	t.Helper()
	w = fiRepo(t)
	remote := t.TempDir()
	gitFI(t, remote, "init", "-q", "--bare")
	gitFI(t, w.Root, "remote", "add", "origin", remote)
	gitFI(t, w.Root, "push", "-q", "origin", "main")

	task = &Task{Seq: 1, Slug: "split", Project: "core"}
	branch := TaskBranch(task)

	gitFI(t, w.Root, "checkout", "-q", "-b", branch)
	if err := os.WriteFile(filepath.Join(w.Root, "one.txt"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFI(t, w.Root, "add", "-A")
	gitFI(t, w.Root, "commit", "-q", "-m", "part one")
	gitFI(t, w.Root, "push", "-q", "origin", branch)
	gitFI(t, w.Root, "fetch", "-q", "origin")

	// Part one merges and is pushed, so BOTH trunk refs contain it.
	gitFI(t, w.Root, "checkout", "-q", "main")
	gitFI(t, w.Root, "merge", "-q", "--no-ff", "-m", "merge part one", branch)
	gitFI(t, w.Root, "push", "-q", "origin", "main")
	gitFI(t, w.Root, "fetch", "-q", "origin")

	// Part two — the deliverable — never leaves the machine.
	gitFI(t, w.Root, "checkout", "-q", branch)
	if err := os.WriteFile(filepath.Join(w.Root, "two.txt"), []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitFI(t, w.Root, "add", "-A")
	gitFI(t, w.Root, "commit", "-q", "-m", "part two - the deliverable")
	localTip = strings.TrimSpace(runOut(t, w.Root, "rev-parse", "HEAD"))
	gitFI(t, w.Root, "checkout", "-q", "main")

	// The premise, asserted: the refs disagree and the deliverable is outside
	// trunk. Without both, the test below proves nothing.
	origin := strings.TrimSpace(runOut(t, w.Root, "rev-parse", "refs/remotes/origin/"+branch))
	if origin == localTip {
		t.Fatalf("setup: the branch refs must disagree, both at %s", origin)
	}
	if out, err := runOutErr(w.Root, "merge-base", "--is-ancestor", localTip, "refs/heads/main"); err == nil {
		t.Fatalf("setup: the deliverable must NOT be in trunk, got %s", out)
	}
	return w, task, localTip
}

// TestCheckLandedRefusesToCertifyAnUnmergedDeliverable is the regression.
func TestCheckLandedRefusesToCertifyAnUnmergedDeliverable(t *testing.T) {
	w, task, _ := splitBranchRepo(t)
	if got, _ := CheckLanded(w, task, "main"); got == LandingLanded {
		t.Fatal("a branch whose local tip is not in trunk must never be certified landed")
	}
}

// TestResolveBranchRefsReturnsBothWhenTheyDisagree pins the mechanism, so a
// future change that reintroduces the single-ref preference fails here with a
// clearer message than the behaviour test alone would give.
func TestResolveBranchRefsReturnsBothWhenTheyDisagree(t *testing.T) {
	w, task, localTip := splitBranchRepo(t)
	_, shas := ResolveBranchRefs(w, task)
	if len(shas) != 2 {
		t.Fatalf("both refs exist and disagree, so both must be returned, got %v", shas)
	}
	var found bool
	for _, s := range shas {
		if s == localTip {
			found = true
		}
	}
	if !found {
		t.Fatalf("the local tip %s carries the deliverable and must be among %v", localTip, shas)
	}
}

// TestCheckLandedStillCertifiesFullyMergedWork stops the fix above from
// degenerating into "never landed": once the deliverable really is in trunk,
// the answer must be Landed, or ship would stamp every task it merged as
// unlanded — the bug at the top of this file, in reverse.
func TestCheckLandedStillCertifiesFullyMergedWork(t *testing.T) {
	w, task, _ := splitBranchRepo(t)
	branch := TaskBranch(task)
	gitFI(t, w.Root, "merge", "-q", "--no-ff", "-m", "merge part two", branch)

	if got, _ := CheckLanded(w, task, "main"); got != LandingLanded {
		t.Fatalf("work fully merged into trunk must read as landed, got %v", got)
	}
}

// TestCheckLandedReportsNoBranchWhenThereIsNone: work committed straight to
// trunk, a docs task, a record task. Nothing to contradict, and it must not be
// reported as unlanded.
func TestCheckLandedReportsNoBranchWhenThereIsNone(t *testing.T) {
	w := fiRepo(t)
	got, _ := CheckLanded(w, &Task{Seq: 99, Slug: "branchless", Project: "core"}, "main")
	if got != LandingNoBranch {
		t.Fatalf("a task with no branch must read as NoBranch, got %v", got)
	}
}
