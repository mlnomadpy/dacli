package execution

import (
	"strings"
	"testing"
)

// A worktree child's brief must resolve TWO destinations, not conflate them
// (task 260): its CODE lands on its own branch, but dacli's workspace state
// (identity, task check, notes, findings, commit event crumb) deliberately
// resolves to the SHARED main workspace, per the workspace.Find redirect. The
// old preamble said only "dacli commit operates HERE", which reads as "my
// records land on my branch" — the exact false belief that makes an agent cd
// to main to 'fix' where its reports went. Spawn must SAY the state is shared.
func TestWorktreePreambleStatesSharedWorkspaceResolution(t *testing.T) {
	const wt = "/repo/.dacli/worktrees/core-260-x"
	got := worktreePreamble(wt)

	if !strings.Contains(got, wt) {
		t.Errorf("preamble does not name the actual worktree path %q:\n%s", wt, got)
	}

	// It must still keep code/edits in the worktree (the sandbox-signal fix).
	for _, want := range []string{"ISOLATED WORKTREE", "relative to it", "THIS branch"} {
		if !strings.Contains(got, want) {
			t.Errorf("preamble no longer keeps edits in the worktree — missing %q:\n%s", want, got)
		}
	}

	// The new, load-bearing part: it must tell the agent that dacli's workspace
	// STATE is shared with the main workspace, not per-branch — otherwise the
	// agent is surprised its task check / notes never reach its branch.
	for _, want := range []string{"MAIN workspace", "task check", "shared"} {
		if !strings.Contains(got, want) {
			t.Errorf("preamble does not tell the agent workspace state is shared — missing %q:\n%s", want, got)
		}
	}

	// And it must warn against the concrete footgun: cd-ing to main to 'fix' it.
	if !strings.Contains(got, "cd") || !strings.Contains(got, "main") {
		t.Errorf("preamble does not warn against cd-ing to the main checkout:\n%s", got)
	}
}
