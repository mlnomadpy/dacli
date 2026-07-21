package cli

import (
	"strings"
	"testing"
)

// The P1 acceptance test, verbatim from PROPOSALS.md: a lesson recorded in
// project A demonstrably appears in a brief assembled in project B. This is
// the compounding loop — every other feature's value is linear in use; this
// one's is superlinear.
func TestLessonCrossesProjects(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "Alpha", "--slug", "alpha", "--goal", "a")
	run(t, dir, 0, "project", "add", "Beta", "--slug", "beta", "--goal", "b")
	run(t, dir, 0, "task", "add", "Alpha task one", "--project", "alpha", "--accept", "a")
	run(t, dir, 0, "task", "add", "Beta task one", "--project", "beta", "--accept", "b")

	// A project-scoped finding in alpha stays in alpha. (Seq numbers are
	// per-project — both tasks are 001 — so refs here must be slugs.)
	run(t, dir, 0, "note", "add", "finding", "Alpha-local detail",
		"--project", "alpha", "--body", "only alpha cares about this")
	briefB := run(t, dir, 0, "context", "beta-task-one")
	if strings.Contains(briefB, "only alpha cares") {
		t.Fatal("project-scoped note leaked across projects")
	}

	// A workspace-scoped lesson in alpha reaches beta's brief, attributed
	// and quote-fenced (data, not instructions).
	run(t, dir, 0, "retro", "alpha",
		"--well", "the batch-path audit before estimating saved a turn",
		"--bad", "estimates ran 2x hot",
		"--improve", "always audit write paths before estimating ledger work",
		"--scope", "workspace")

	briefB = run(t, dir, 0, "context", "beta-task-one")
	if !strings.Contains(briefB, "## Lessons from other projects") {
		t.Fatalf("lessons section missing from cross-project brief:\n%s", briefB)
	}
	if !strings.Contains(briefB, "audit write paths before estimating") {
		t.Errorf("lesson content missing:\n%s", briefB)
	}
	if !strings.Contains(briefB, "from alpha") {
		t.Errorf("lesson not attributed to its source project:\n%s", briefB)
	}

	// And alpha's own brief does NOT repeat it in the lessons section — the
	// cross-project channel is strictly for other projects.
	briefA := run(t, dir, 0, "context", "alpha-task-one")
	if strings.Contains(briefA, "Lessons from other projects") {
		t.Errorf("own-project lesson echoed back through the cross-project channel:\n%s", briefA)
	}
}
