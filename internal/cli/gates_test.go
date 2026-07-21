package cli

import (
	"strings"
	"testing"
)

// Task 001's acceptance criteria, as tests: filled-not-present predicates,
// solo default with zero gates, and refusal-with-unmet-list at exit 3.
func TestStageGates(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")

	// solo by default: no template, no gates, nothing to advance.
	run(t, dir, 0, "project", "add", "Free project", "--slug", "free", "--goal", "whatever moves")
	if out := run(t, dir, 0, "stage", "free"); !strings.Contains(out, "no template (solo): no gates") {
		t.Errorf("solo default wrong:\n%s", out)
	}

	// A gated project, born with a deliberately unfilled goal.
	out := run(t, dir, 0, "project", "add", "Gated project", "--slug", "g",
		"--goal", "TBD", "--template", "standard")
	if !strings.Contains(out, "template standard attached (stage: define)") {
		t.Fatalf("template not attached:\n%s", out)
	}

	// FILLED, not present: "TBD" fails with the reason named.
	st := run(t, dir, 0, "stage", "g")
	if !strings.Contains(st, "✗ project sections filled: Goal") || !strings.Contains(st, "placeholder TBD") {
		t.Errorf("TBD goal not caught as unfilled:\n%s", st)
	}

	// stage advance refuses at exit 3 with the full unmet list.
	refusal := run(t, dir, 3, "stage", "advance", "g")
	for _, want := range []string{"gate closed", "placeholder TBD", "glossary has ≥2 terms", "decision(s) with a rejection", "do not retry"} {
		if !strings.Contains(refusal, want) {
			t.Errorf("refusal missing %q:\n%s", want, refusal)
		}
	}

	// An ambiguous goal is exactly as empty as TBD — the SPM linter is the
	// gate's content check. (Rewrite project.md via a fresh project.)
	run(t, dir, 0, "project", "add", "Vague project", "--slug", "v",
		"--goal", "improve the system and handle all the edge cases properly", "--template", "standard")
	st = run(t, dir, 0, "stage", "v")
	if !strings.Contains(st, "ambiguous at major severity") || !strings.Contains(st, "as empty as TBD") {
		t.Errorf("ambiguous goal not caught:\n%s", st)
	}

	// Now a project that can actually pass: filled goal, glossary, decision.
	run(t, dir, 0, "project", "add", "Real project", "--slug", "r",
		"--goal", "One write path into balances, verified by the reconciliation suite.",
		"--template", "standard")
	run(t, dir, 0, "glossary", "r", "--term", "balance", "--def", "the authoritative row")
	run(t, dir, 0, "glossary", "r", "--term", "shim", "--def", "the single write wrapper")
	run(t, dir, 0, "note", "add", "decision", "Writes stay synchronous", "--project", "r",
		"--rejected", "async queue", "--because", "reconciliation cost exceeds the win")

	out = run(t, dir, 0, "stage", "advance", "r")
	if !strings.Contains(out, "advanced to stage build") {
		t.Fatalf("define gate did not open:\n%s", out)
	}
	// Passing a gate narrows the Cone on the project itself.
	if show := run(t, dir, 0, "project", "show", "r"); !strings.Contains(show, "stage: approach") {
		t.Errorf("cone not narrowed on advance:\n%s", show)
	}

	// build gate: tasks need acceptance + estimates.
	run(t, dir, 0, "task", "add", "Build the shim", "--project", "r", "--priority", "must", "--accept", "suite green")
	refusal = run(t, dir, 3, "stage", "advance", "r")
	if !strings.Contains(refusal, "three-point estimate") {
		t.Errorf("estimate gate missing:\n%s", refusal)
	}
	// Cannot fix an estimate in place yet, so prove the gate opens on a
	// compliant project: everything estimated from birth.
	run(t, dir, 0, "project", "add", "Clean project", "--slug", "c",
		"--goal", "One write path into balances, verified by the reconciliation suite.",
		"--template", "standard")
	run(t, dir, 0, "glossary", "c", "--term", "a", "--def", "x")
	run(t, dir, 0, "glossary", "c", "--term", "b", "--def", "y")
	run(t, dir, 0, "note", "add", "decision", "Writes stay synchronous", "--project", "c",
		"--rejected", "async queue", "--because", "cost exceeds win")
	run(t, dir, 0, "task", "add", "Ship the shim", "--project", "c", "--priority", "must",
		"--estimate", "1,2,3", "--accept", "suite green")
	run(t, dir, 0, "stage", "advance", "c") // define → build
	run(t, dir, 0, "stage", "advance", "c") // build → ship
	refusal = run(t, dir, 3, "stage", "advance", "c")
	if !strings.Contains(refusal, "every must task is done") || !strings.Contains(refusal, "retro is recorded") {
		t.Errorf("ship gate wrong:\n%s", refusal)
	}
	run(t, dir, 0, "task", "claim", "ship-the-shim")
	run(t, dir, 0, "task", "check", "ship-the-shim", "--all")
	run(t, dir, 0, "task", "done", "ship-the-shim")
	run(t, dir, 0, "retro", "c", "--well", "gates found the vague goal early")
	out = run(t, dir, 0, "stage", "advance", "c")
	if !strings.Contains(out, "template complete") {
		t.Errorf("final gate did not complete:\n%s", out)
	}
}

func TestTemplateListShowVendor(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")

	out := run(t, dir, 0, "template", "list")
	for _, want := range []string{"solo", "standard", "research-paper", "zero gates; most work should use this"} {
		if !strings.Contains(out, want) {
			t.Errorf("template list missing %q:\n%s", want, out)
		}
	}
	if show := run(t, dir, 0, "template", "show", "standard"); !strings.Contains(show, "## stage: define") {
		t.Errorf("template show wrong:\n%s", show)
	}
	if v := run(t, dir, 0, "template", "add", "standard"); !strings.Contains(v, "vendored to") {
		t.Errorf("vendor failed:\n%s", v)
	}
	// Vendored copy now reports as workspace-origin.
	if out := run(t, dir, 0, "template", "list"); !strings.Contains(out, "workspace") {
		t.Errorf("vendored origin not shown:\n%s", out)
	}
	run(t, dir, 1, "template", "add", "standard") // double-vendor refused
}
