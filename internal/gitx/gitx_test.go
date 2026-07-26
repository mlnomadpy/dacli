package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// repoOnMainWithBranch builds a repo on main with a base commit plus a `feature`
// branch that changes code.txt, then leaves main checked out.
func repoOnMainWithBranch(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "x@x")
	git(t, dir, "config", "user.name", "x")
	git(t, dir, "checkout", "-q", "-b", "main")
	write(t, dir, "code.txt", "base\n")
	write(t, dir, ".dacli/tasks/open/1-a.md", "base task\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")

	git(t, dir, "checkout", "-q", "-b", "feature")
	write(t, dir, "code.txt", "base\nfeature line\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "feature")
	git(t, dir, "checkout", "-q", "main")
	return dir
}

// A dirty .dacli (the state `dacli accept` leaves behind by moving a tracked
// task file between status folders) must NOT block a code-branch merge — that
// is what once made `dacli ship` fail to integrate right after accepting.
func TestMergeToleratesDirtyDacli(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	// Simulate accept: move the tracked task file (dirty, tracked deletion) and
	// drop the new copy in done/ — exactly what MoveTask does.
	if err := os.Rename(filepath.Join(dir, ".dacli/tasks/open/1-a.md"), filepath.Join(dir, ".dacli/tasks/done/1-a.md")); err != nil {
		// done dir may not exist yet
		write(t, dir, ".dacli/tasks/done/1-a.md", "base task\n")
		_ = os.Remove(filepath.Join(dir, ".dacli/tasks/open/1-a.md"))
	}
	if IsClean(dir) {
		t.Fatal("precondition: tree should be dirty (.dacli task moved)")
	}
	conflicts, err := Merge(dir, "feature", "merge feature")
	if err != nil {
		t.Fatalf("merge refused despite only .dacli being dirty: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", conflicts)
	}
	if got := git(t, dir, "log", "-1", "--format=%s"); got == "" {
		t.Fatal("no merge commit")
	}
}

// A dirty CODE file still blocks the merge — it could be clobbered — and the
// failure is a real error, never a mislabeled conflict.
func TestMergeRefusesDirtyCode(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	write(t, dir, "code.txt", "base\nlocal uncommitted edit\n")
	conflicts, err := Merge(dir, "feature", "merge feature")
	if err == nil {
		t.Fatal("merge should refuse with a dirty code tree")
	}
	if len(conflicts) != 0 {
		t.Fatalf("a dirty tree is not a conflict; got conflicts %v", conflicts)
	}
}

// A merge that fails for a NON-conflict reason (here: a branch that does not
// exist) returns a real error with no conflicted files, so the caller can tell
// it apart from a genuine conflict and not swallow it to success.
func TestMergeMissingBranchIsErrorNotConflict(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	conflicts, err := Merge(dir, "does-not-exist", "merge ghost")
	if err == nil {
		t.Fatal("merging a missing branch should error")
	}
	if len(conflicts) != 0 {
		t.Fatalf("missing branch is not a conflict; got %v", conflicts)
	}
}

// twoClonesOfBareOrigin sets up a bare origin plus two working clones of it
// (a and b) both tracking `main` — standing in for "the loop's own local
// checkout" (a) and "whatever landed the async merge" (b, e.g. GitHub's own
// merge of a fixer PR via `gh pr merge --auto`).
func twoClonesOfBareOrigin(t *testing.T) (origin, a, b string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	origin = filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", origin).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	seed := t.TempDir()
	git(t, seed, "init", "-q", "-b", "main")
	git(t, seed, "config", "user.email", "x@x")
	git(t, seed, "config", "user.name", "x")
	write(t, seed, "code.txt", "base\n")
	git(t, seed, "add", "-A")
	git(t, seed, "commit", "-q", "-m", "base")
	git(t, seed, "remote", "add", "origin", origin)
	git(t, seed, "push", "-q", "-u", "origin", "main")

	a = filepath.Join(t.TempDir(), "a")
	b = filepath.Join(t.TempDir(), "b")
	if out, err := exec.Command("git", "clone", "-q", origin, a).CombinedOutput(); err != nil {
		t.Fatalf("git clone a: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "clone", "-q", origin, b).CombinedOutput(); err != nil {
		t.Fatalf("git clone b: %v\n%s", err, out)
	}
	for _, dir := range []string{a, b} {
		git(t, dir, "config", "user.email", "x@x")
		git(t, dir, "config", "user.name", "x")
	}
	return origin, a, b
}

// A local checkout (a) that has simply fallen behind origin — b pushed a new
// commit and a never re-fetched — must be fast-forwarded to match origin
// exactly, discarding nothing (there is nothing local to discard).
func TestFastForwardAdvancesLocalToOrigin(t *testing.T) {
	_, a, b := twoClonesOfBareOrigin(t)
	write(t, b, "landed.txt", "async auto-merge\n")
	git(t, b, "add", "-A")
	git(t, b, "commit", "-q", "-m", "landed via gh pr merge --auto")
	git(t, b, "push", "-q", "origin", "main")

	if out, err := FastForward(a, "main"); err != nil {
		t.Fatalf("FastForward: %v (%s)", err, out)
	}
	got := strings.TrimSpace(git(t, a, "rev-parse", "HEAD"))
	want := strings.TrimSpace(git(t, b, "rev-parse", "HEAD"))
	if got != want {
		t.Fatalf("local main not fast-forwarded to origin: got %s want %s", got, want)
	}
}

// A local checkout that DIVERGED from origin (both gained their own unique
// commit — the record commit locally, an async auto-merge on origin) must
// never be silently clobbered by FastForward — --ff-only refuses, and the
// caller (never this function) decides whether to rebase (PushSync does).
func TestFastForwardRefusesOnDivergedHistories(t *testing.T) {
	_, a, b := twoClonesOfBareOrigin(t)

	write(t, b, "landed.txt", "async auto-merge\n")
	git(t, b, "add", "-A")
	git(t, b, "commit", "-q", "-m", "landed via gh pr merge --auto")
	git(t, b, "push", "-q", "origin", "main")

	write(t, a, "local.txt", "not yet pushed\n")
	git(t, a, "add", "-A")
	git(t, a, "commit", "-q", "-m", "local record commit")
	before := strings.TrimSpace(git(t, a, "rev-parse", "HEAD"))

	if _, err := FastForward(a, "main"); err == nil {
		t.Fatal("expected FastForward to refuse diverged histories rather than clobber local work")
	}
	after := strings.TrimSpace(git(t, a, "rev-parse", "HEAD"))
	if after != before {
		t.Fatalf("FastForward must never move HEAD on refusal: before %s after %s", before, after)
	}
}

// The record-push case: origin gained a commit (the async auto-merge) AND
// the local checkout has its own new commit to push (the .dacli record).
// PushSync must fetch, rebase the local commit onto origin, and land both.
func TestPushSyncRebasesOntoOriginOnNonFastForwardRejection(t *testing.T) {
	_, a, b := twoClonesOfBareOrigin(t)

	write(t, b, "landed.txt", "async auto-merge\n")
	git(t, b, "add", "-A")
	git(t, b, "commit", "-q", "-m", "landed via gh pr merge --auto")
	git(t, b, "push", "-q", "origin", "main")

	write(t, a, "record.txt", "workspace record\n")
	git(t, a, "add", "-A")
	git(t, a, "commit", "-q", "-m", "ship: record workspace")

	out, err := Push(a, "main")
	if err == nil {
		t.Fatalf("test setup: expected the bare push to be rejected non-fast-forward, got success: %s", out)
	}
	if !isNonFastForward(out) {
		t.Fatalf("test setup: expected a non-fast-forward rejection, got: %s", out)
	}

	if out, err := PushSync(a, "main"); err != nil {
		t.Fatalf("PushSync: %v (%s)", err, out)
	}

	log := git(t, a, "log", "--format=%s", "origin/main")
	if !strings.Contains(log, "landed via gh pr merge --auto") || !strings.Contains(log, "ship: record workspace") {
		t.Fatalf("expected both the async merge and the local record commit on origin/main after PushSync, got:\n%s", log)
	}
}

// A genuine content conflict between the local commit and what landed on
// origin must abort the rebase cleanly and return the ORIGINAL push error —
// never leave the tree mid-rebase.
func TestPushSyncAbortsCleanlyOnRebaseConflict(t *testing.T) {
	_, a, b := twoClonesOfBareOrigin(t)

	write(t, b, "code.txt", "base\nconflicting change from origin\n")
	git(t, b, "add", "-A")
	git(t, b, "commit", "-q", "-m", "origin change")
	git(t, b, "push", "-q", "origin", "main")

	write(t, a, "code.txt", "base\nconflicting change from local\n")
	git(t, a, "add", "-A")
	git(t, a, "commit", "-q", "-m", "local change")

	if _, err := PushSync(a, "main"); err == nil {
		t.Fatal("expected PushSync to fail on a genuine rebase conflict")
	}
	status := git(t, a, "status", "--porcelain=v1")
	if strings.Contains(status, "U ") || strings.Contains(status, "UU") {
		t.Fatalf("PushSync left unresolved conflict markers staged, tree not cleanly aborted: %s", status)
	}
	rebaseDir := filepath.Join(a, ".git", "rebase-merge")
	if _, err := os.Stat(rebaseDir); err == nil {
		t.Fatal("PushSync left the repo mid-rebase after a conflict")
	}
}

func TestIsCleanExcept(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	if !IsCleanExcept(dir, ".dacli") {
		t.Fatal("clean tree should be clean")
	}
	write(t, dir, ".dacli/tasks/open/2-b.md", "new\n")
	// Untracked .dacli file is invisible to --untracked-files=no, still clean.
	if !IsCleanExcept(dir, ".dacli") {
		t.Fatal(".dacli-only change should be clean-except-.dacli")
	}
	write(t, dir, "code.txt", "changed\n")
	if IsCleanExcept(dir, ".dacli") {
		t.Fatal("a dirty tracked code file must not be clean-except-.dacli")
	}
}

func TestIsAncestorTrueWhenMerged(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	if _, err := Merge(dir, "feature", "merge feature"); err != nil {
		t.Fatalf("merge: %v", err)
	}
	ok, err := IsAncestor(dir, "feature", "main")
	if err != nil {
		t.Fatalf("IsAncestor: %v", err)
	}
	if !ok {
		t.Fatal("feature was merged into main — should report ancestor")
	}
}

func TestIsAncestorFalseWhenNotMerged(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	ok, err := IsAncestor(dir, "feature", "main")
	if err != nil {
		t.Fatalf("IsAncestor: %v", err)
	}
	if ok {
		t.Fatal("feature has not been merged — should not report ancestor")
	}
}

func TestIsAncestorErrorsOnUnknownRef(t *testing.T) {
	dir := repoOnMainWithBranch(t)
	if _, err := IsAncestor(dir, "does-not-exist", "main"); err == nil {
		t.Fatal("an unknown ref should be a real error, not a false negative")
	}
}
