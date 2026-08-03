package orchestration

// Stack-aware loop role defaults (dacli 192). The loop used to hand every
// project the constants "fixer" and "go-auditor" — a Python app was reviewed by
// a role named for a language it does not contain. These tests pin both halves:
// a recorded stack moves the default onto a roster role that exists, and a
// project with nothing recorded stays on the pre-192 constants exactly.

import (
	"testing"

	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// recordStack writes the stack the way `dacli new` writes it, so this test
// exercises the same prose the real command produces rather than a shape
// invented for the test.
func recordStack(t *testing.T, w *workspace.Workspace, slug, body string) {
	t.Helper()
	p, err := store.LoadProject(w, slug)
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.SetSection("Constraints", body)
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
}

func rosterLookup(w *workspace.Workspace) func(string) bool {
	return func(name string) bool { _, ok := store.LoadRole(w, name); return ok }
}

func TestLoopRolesFollowRecordedStack(t *testing.T) {
	w := loopEnv(t)
	recordStack(t, w, "p", "Stack: Python. Build with `python -m build`; test with `pytest`. A task in this project is done only when both exit 0.\n")
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "python-auditor", Summary: "Audits Python for correctness and style.", Kind: "reviewer",
	}); err != nil {
		t.Fatal(err)
	}

	s := loopStack(w, "p")
	if s.Key != "python" {
		t.Fatalf("stack not read back from the project doc: %+v", s)
	}
	exists := rosterLookup(w)
	if got := prompts.RoleFor(s, "auditor", "go-auditor", exists); got != "python-auditor" {
		t.Errorf("review role = %q, want python-auditor", got)
	}
	// No python-fixer in the roster, so the impl default holds — the loop must
	// never spawn a role name that does not exist.
	if got := prompts.RoleFor(s, "fixer", "fixer", exists); got != "fixer" {
		t.Errorf("impl role = %q, want fixer", got)
	}
}

func TestLoopRolesUnchangedWithoutRecordedStack(t *testing.T) {
	w := loopEnv(t) // loopEnv's project records no stack, like every pre-192 one
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "python-auditor", Summary: "Audits Python for correctness and style.", Kind: "reviewer",
	}); err != nil {
		t.Fatal(err)
	}

	s := loopStack(w, "p")
	if s.Recorded() {
		t.Fatalf("a stackless project produced a stack: %+v", s)
	}
	exists := rosterLookup(w)
	if got := prompts.RoleFor(s, "auditor", "go-auditor", exists); got != "go-auditor" {
		t.Errorf("review role = %q, want go-auditor (pre-192 default)", got)
	}
	if got := prompts.RoleFor(s, "fixer", "fixer", exists); got != "fixer" {
		t.Errorf("impl role = %q, want fixer (pre-192 default)", got)
	}
	// A project that does not exist at all is the same answer, not a panic.
	if missing := loopStack(w, "no-such-project"); missing.Recorded() {
		t.Errorf("missing project produced a stack: %+v", missing)
	}
}
