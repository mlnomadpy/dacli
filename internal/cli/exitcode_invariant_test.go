package cli

import "testing"

// TestExitCodeContract is an INVARIANT test over the documented contract
// (ARCHITECTURE § 4): 2 usage, 3 refused-by-policy, 4 not found, 1 everything
// else. Agents branch on these codes without parsing stderr, and the 1/3
// distinction is the load-bearing one — retrying a refusal is the loop a
// supervisor must never enter.
//
// It is a table rather than per-command tests for the same reason the
// capability invariants are: the drift found by audit was never one command
// being wrong on purpose, it was a new path returning a bare error where the
// contract promised a specific code. A table fails when a code changes,
// whoever changed it (dacli 202).
func TestExitCodeContract(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a genuine goal for this project")
	run(t, dir, 0, "task", "add", "a real task", "--project", "p", "--accept", "it works")

	for _, tc := range []struct {
		name string
		want int
		args []string
	}{
		// 2 — usage: the caller's invocation is malformed. Fix the command line.
		{"unknown flag", 2, []string{"task", "list", "--nosuchflag", "x"}},
		{"missing required flag", 2, []string{"note", "add", "decision", "a title"}},
		{"unknown note kind", 2, []string{"note", "add", "findings", "t", "--project", "p"}},
		{"no positional where one is required", 2, []string{"task", "show"}},

		// 4 — not found: the reference is well-formed but names nothing.
		{"missing task", 4, []string{"task", "show", "999"}},
		{"missing task by slug", 4, []string{"task", "show", "no-such-task"}},

		// 3 — refused by policy: a real answer. NEVER retry.
		{"task done with unmet acceptance", 3, []string{"task", "done", "001"}},

		// 0 — the happy paths, so the table also proves it is not just
		// asserting that everything fails.
		{"task list", 0, []string{"task", "list", "--project", "p"}},
		{"task show", 0, []string{"task", "show", "001"}},
		{"doctor", 0, []string{"doctor"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run(t, dir, tc.want, tc.args...) // run() fails the test on a mismatch
		})
	}
}

// The distinction that matters most, stated on its own so a regression names
// itself: a refusal is an ANSWER (3) and a supervisor must not retry it, while
// a not-found (4) means the caller asked about something that does not exist.
// Collapsing either into the generic 1 teaches agents to retry work that will
// never succeed.
func TestRefusalAndNotFoundAreDistinctFromGenericFailure(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a genuine goal for this project")
	run(t, dir, 0, "task", "add", "a real task", "--project", "p", "--accept", "it works")

	if out := run(t, dir, 3, "task", "done", "001"); out == "" {
		t.Error("a refusal must explain itself, not just exit 3")
	}
	if out := run(t, dir, 4, "task", "show", "12345"); out == "" {
		t.Error("a not-found must name what was not found")
	}
}
