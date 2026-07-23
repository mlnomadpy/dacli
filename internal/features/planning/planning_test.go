package planning

import (
	"bytes"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// taskAddEnv builds a workspace with one project and returns a Ctx ready to
// drive cmdTaskAdd directly. DACLI_AGENT is cleared so the acting identity is
// root regardless of who runs the suite.
func taskAddEnv(t *testing.T) (*workspace.Workspace, *clikit.Ctx) {
	t.Helper()
	t.Setenv("DACLI_AGENT", "")
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	return w, &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
}

// TestTaskAddRefusesNearDuplicateOfOpenTask reproduces the dacli task 116
// incident: a review auditor re-filing an already-queued issue under
// slightly different wording must be refused (exit 3), not silently allowed
// to create backlog churn.
func TestTaskAddRefusesNearDuplicateOfOpenTask(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{"charge idle-cycle review spawns to the token window", "--project", "p"}); err != nil {
		t.Fatalf("first add: %v", err)
	}

	err := cmdTaskAdd(ctx, []string{"charge idle-cycle reviewer tokens to the --window-tokens budget", "--project", "p"})
	if err == nil {
		t.Fatal("near-duplicate title was accepted, want refusal")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refusal)", clikit.ExitCode(err))
	}

	ts, lerr := store.ListTasks(w, "p", "")
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(ts) != 1 {
		t.Errorf("project has %d tasks after refused dup, want 1", len(ts))
	}
}

// TestTaskAddForceOverridesDedup confirms --force is the explicit, loud
// override — same shape as spawn/accept's --force — rather than a dead end.
func TestTaskAddForceOverridesDedup(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{"charge idle-cycle review spawns to the token window", "--project", "p"}); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := cmdTaskAdd(ctx, []string{"charge idle-cycle reviewer tokens to the --window-tokens budget", "--project", "p", "--force"}); err != nil {
		t.Fatalf("forced add: %v", err)
	}

	ts, err := store.ListTasks(w, "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 {
		t.Errorf("project has %d tasks after forced dup, want 2", len(ts))
	}
}

// TestTaskAddAllowsUnrelatedTitles is the control: ordinary, distinct backlog
// titles must never be blocked by the dedup guard.
func TestTaskAddAllowsUnrelatedTitles(t *testing.T) {
	w, ctx := taskAddEnv(t)
	if err := cmdTaskAdd(ctx, []string{"fix flaky retry timer in the spawn watchdog", "--project", "p"}); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := cmdTaskAdd(ctx, []string{"document the SPM glossary term for slack", "--project", "p"}); err != nil {
		t.Fatalf("second add: %v", err)
	}

	ts, err := store.ListTasks(w, "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 {
		t.Errorf("project has %d tasks, want 2", len(ts))
	}
}
