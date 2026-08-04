package store

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// gitWorkspace inits a git repo, a dacli workspace and a project inside it, and
// commits the lot on the default branch so later branches share a common base.
// It reuses the package-level git() helper (version_test.go). Skips when git is
// absent — the collision it guards is a property of git history.
func gitWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	w, err := workspace.Init(dir, "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")
	return w
}

// A task filed on one branch must not hand out a seq another, unmerged branch
// already committed. Both branches are cut from the same base, so each one's
// WORKING TREE sees the same max seq — the exact setup that, allocating against
// the working tree alone, gives both the same NNN. When the branches merge, two
// different tasks then share one number and `dacli <NNN>` cannot resolve it
// (dacli 251, the cross-branch twin of the 209 concurrent collision). Fixed by
// clearing the ceiling of every seq committed on ANY ref, so the second branch
// sees the first branch's task even though its own tree does not.
func TestCreateTaskNeverReusesASeqFromAnotherBranch(t *testing.T) {
	w := gitWorkspace(t)
	dir := w.Root

	// Branch A files the first task and commits it.
	git(t, dir, "checkout", "-q", "-b", "branchA")
	a, err := CreateTask(w, "a-root", "core", "task on A", TaskOpts{})
	if err != nil {
		t.Fatalf("create on A: %v", err)
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "task on A")

	// Branch B is cut from the shared base, so its working tree does NOT contain
	// A's task file — allocating against the tree alone would reuse A's seq.
	git(t, dir, "checkout", "-q", "main")
	git(t, dir, "checkout", "-q", "-b", "branchB")
	b, err := CreateTask(w, "a-root", "core", "task on B", TaskOpts{})
	if err != nil {
		t.Fatalf("create on B: %v", err)
	}

	if a.Seq == b.Seq {
		t.Fatalf("branch B reused branch A's seq %d — the two tasks now share a reference", a.Seq)
	}
	if b.Seq != a.Seq+1 {
		t.Fatalf("branch B got seq %d, want %d (next after A's committed %d)", b.Seq, a.Seq+1, a.Seq)
	}
}

// A seq committed on a branch and then DELETED there is still spent: reusing it
// would resurrect an ambiguous reference the moment history is examined. The
// git-wide ceiling walks history, not just branch tips, so a renamed-away or
// removed task file keeps its number reserved.
func TestCreateTaskNeverReusesADeletedBranchSeq(t *testing.T) {
	w := gitWorkspace(t)
	dir := w.Root

	git(t, dir, "checkout", "-q", "-b", "branchA")
	a, err := CreateTask(w, "a-root", "core", "ephemeral", TaskOpts{})
	if err != nil {
		t.Fatalf("create on A: %v", err)
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "add")
	git(t, dir, "rm", "-q", a.Path)
	git(t, dir, "commit", "-q", "-m", "remove")

	git(t, dir, "checkout", "-q", "main")
	b, err := CreateTask(w, "a-root", "core", "fresh", TaskOpts{})
	if err != nil {
		t.Fatalf("create on main: %v", err)
	}
	if b.Seq == a.Seq {
		t.Fatalf("reused seq %d from a task deleted on branchA", a.Seq)
	}
}

// A pre-existing collision — two different tasks that already share one seq,
// the wreckage a merge of two colliding branches leaves — must be surfaced for
// reconciliation, not left buried behind an "ambiguous ref" at the point of
// use. CollidedSeqs names both so an owner can renumber one.
func TestCollidedSeqsSurfacesDistinctTasksSharingASeq(t *testing.T) {
	w := indexWorkspace(t)

	// Two genuinely different tasks land on the same NNN (the post-merge state).
	first, err := CreateTask(w, "a-root", "core", "first winner", TaskOpts{})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	twin, err := CreateTask(w, "a-root", "core", "second winner", TaskOpts{})
	if err != nil {
		t.Fatalf("twin: %v", err)
	}
	// Force the collision on disk: rename the twin's file to the first's seq.
	collided := strings.Replace(twin.Path, fmt.Sprintf("%03d-", twin.Seq), fmt.Sprintf("%03d-", first.Seq), 1)
	if err := os.Rename(twin.Path, collided); err != nil {
		t.Fatalf("rename: %v", err)
	}

	cols, err := CollidedSeqs(w)
	if err != nil {
		t.Fatalf("collided: %v", err)
	}
	if len(cols) != 1 {
		t.Fatalf("got %d collisions, want 1: %+v", len(cols), cols)
	}
	if cols[0].Seq != first.Seq || len(cols[0].Slugs) != 2 {
		t.Fatalf("collision = seq %d with %d slugs, want seq %d with 2", cols[0].Seq, len(cols[0].Slugs), first.Seq)
	}

	// A clean workspace (one task per seq) reports nothing.
	clean := indexWorkspace(t)
	if _, err := CreateTask(clean, "a-root", "core", "alone", TaskOpts{}); err != nil {
		t.Fatalf("clean create: %v", err)
	}
	if cols, _ := CollidedSeqs(clean); len(cols) != 0 {
		t.Fatalf("clean workspace reported %d collisions", len(cols))
	}
}

// gitTaskSeqCeiling degrades to the working-tree scan (returns 0) when there is
// no git repo, so every non-git test workspace keeps allocating exactly as
// before. Guards against a regression where a git failure would wedge creation.
func TestGitTaskSeqCeilingIsZeroWithoutARepo(t *testing.T) {
	w := indexWorkspace(t) // t.TempDir(), not a git repo
	if got := gitTaskSeqCeiling(w, "core"); got != 0 {
		t.Fatalf("ceiling in a non-git workspace = %d, want 0", got)
	}
	// And creation still works there.
	if _, err := CreateTask(w, "a-root", "core", "no git here", TaskOpts{}); err != nil {
		t.Fatalf("create without git: %v", err)
	}
}
