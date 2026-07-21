package cli

import (
	"strings"
	"testing"
)

// The core ask: don't start implementation while still in discovery. Phase
// gating refuses an implementer role until the project reaches a build phase,
// and the brief tells the agent what work is appropriate now.
func TestPhaseGating(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")

	// A phase-gated project starts in discovery.
	run(t, dir, 0, "project", "add", "Widget", "--slug", "w",
		"--goal", "One clear thing, verified by the acceptance suite.", "--template", "product")
	run(t, dir, 0, "task", "add", "Prototype the widget", "--project", "w", "--accept", "a")

	// Roles with kinds, and a runtime to spawn onto.
	run(t, dir, 0, "role", "add", "researcher", "--kind", "researcher", "--grant", "ro")
	run(t, dir, 0, "role", "add", "builder", "--kind", "implementer", "--grant", "rw")
	run(t, dir, 0, "runtime", "add", "mock", "--binary", "true", "--mode", "stdin")

	// stage shows the phase and its allowed roles.
	st := run(t, dir, 0, "stage", "w")
	if !strings.Contains(st, "phase discovery") || !strings.Contains(st, "researcher") {
		t.Errorf("stage does not show the phase:\n%s", st)
	}

	// The brief tells the agent the phase.
	brief := run(t, dir, 0, "context", "prototype-the-widget")
	if !strings.Contains(brief, "Phase: **discovery**") || !strings.Contains(brief, "work appropriate now: researcher") {
		t.Errorf("brief missing the phase:\n%s", brief)
	}

	// Spawning an IMPLEMENTER during discovery is refused (exit 3).
	refusal := run(t, dir, 3, "spawn", "--task", "prototype-the-widget", "--role", "builder", "--runtime", "mock", "--cooperative")
	if !strings.Contains(refusal, "in the discovery phase") || !strings.Contains(refusal, "implementer role has no work here") {
		t.Errorf("implementer not blocked in discovery:\n%s", refusal)
	}

	// A researcher IS allowed in discovery (mock binary is `true`, so it runs
	// and exits clean).
	ok := run(t, dir, 0, "spawn", "--task", "prototype-the-widget", "--role", "researcher", "--runtime", "mock", "--cooperative")
	if !strings.Contains(ok, "ok in") {
		t.Errorf("researcher blocked in its own phase:\n%s", ok)
	}

	// A role with NO kind opts out of phase gating entirely.
	run(t, dir, 0, "role", "add", "anything", "--grant", "rw", "--summary", "unkinded")
	run(t, dir, 0, "spawn", "--task", "prototype-the-widget", "--role", "anything", "--runtime", "mock", "--cooperative")

	// Advance through the gates (fill each phase's requirements) until build,
	// where the implementer is finally welcome. Discovery needs Goal +
	// Out-of-scope filled + 3 glossary terms.
	// (Goal is filled; add Out of scope by editing is not exposed, so this
	// test asserts the GATE holds the implementer out — advancing fully is
	// covered by TestStageGates. Here we prove the phase gate is real.)
	if strings.Contains(run(t, dir, 0, "stage", "w"), "phase implementation") {
		t.Error("project should still be pre-implementation")
	}
}

// Untemplated (solo) projects are never phase-gated — any role spawns.
func TestSoloProjectUngated(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "Free", "--slug", "f", "--goal", "g")
	run(t, dir, 0, "task", "add", "Do the thing", "--project", "f", "--accept", "a")
	run(t, dir, 0, "role", "add", "builder", "--kind", "implementer", "--grant", "rw")
	run(t, dir, 0, "runtime", "add", "mock", "--binary", "true", "--mode", "stdin")

	// No template → no phase → implementer spawns freely.
	out := run(t, dir, 0, "spawn", "--task", "do-the-thing", "--role", "builder", "--runtime", "mock", "--cooperative")
	if !strings.Contains(out, "ok in") {
		t.Errorf("solo project should not gate by phase:\n%s", out)
	}
}
