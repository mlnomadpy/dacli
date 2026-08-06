package acceptance

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// acceptEnv builds a workspace holding one task owned by a *different* agent —
// the stand-in for a spawned child that has since finished and will never sync.
func acceptEnv(t *testing.T) (*workspace.Workspace, *store.Task, *clikit.Ctx) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if v, ok := os.LookupEnv("DACLI_AGENT"); ok { // act as root
		t.Setenv("DACLI_AGENT", v)
		_ = os.Unsetenv("DACLI_AGENT")
	}
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "x@x"}, {"config", "user.name", "x"}, {"checkout", "-q", "-b", "main"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	w, err := workspace.Init(dir, "a-root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTask(w, "a-deadchild", "p", "Orphaned work", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	return w, tk, &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
}

func TestAcceptWithoutForceLeavesOrphanOpen(t *testing.T) {
	w, tk, ctx := acceptEnv(t)
	ref := fmt.Sprintf("%03d", tk.Seq)
	if err := cmdAccept(ctx, []string{ref}); err != nil {
		t.Fatal(err)
	}
	got, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Fatal("plain accept on another agent's task must propose, not close")
	}
}

func TestAcceptForceReconcilesOrphanedTask(t *testing.T) {
	w, tk, ctx := acceptEnv(t)
	ref := fmt.Sprintf("%03d", tk.Seq)
	if err := cmdAccept(ctx, []string{ref, "--force"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusDone {
		t.Fatalf("--force must close the orphaned task, status=%s", got.Status)
	}
	if got.Owner() != "a-root" {
		t.Fatalf("--force must adopt ownership to root, owner=%s", got.Owner())
	}
}

// TestAcceptRefusesEmptyAcceptance is the dacli 289 regression on the accept
// path: CheckAllAcceptance checks zero boxes and reports success on a task with
// no criteria, so accept would close it with nothing verified. It must refuse
// (exit 3) instead, and leave the task open.
func TestAcceptRefusesEmptyAcceptance(t *testing.T) {
	w, _, ctx := acceptEnv(t)
	// A task the acting root owns, with NO acceptance criteria.
	tk, err := store.CreateTask(w, agentid.RootID, "p", "no criteria", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}

	err = acceptOne(ctx, w, root, tk, "", false, false, false)
	if err == nil {
		t.Fatal("accept closed a task with an empty Acceptance section (dacli 289)")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refused-by-policy)", clikit.ExitCode(err))
	}
	got, err := store.FindTask(w, fmt.Sprintf("%03d", tk.Seq))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Fatal("empty-acceptance task was closed by accept despite the refusal")
	}
}

// TestAcceptAllowUnverifiedClosesEmptyAcceptance confirms the escape hatch on
// the accept path: --allow-unverified closes the criteria-less task and stamps
// the Log UNVERIFIED (dacli 289).
func TestAcceptAllowUnverifiedClosesEmptyAcceptance(t *testing.T) {
	w, _, ctx := acceptEnv(t)
	tk, err := store.CreateTask(w, agentid.RootID, "p", "chore", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}

	if err := acceptOne(ctx, w, root, tk, "", false, false, true); err != nil {
		t.Fatalf("--allow-unverified must close the task: %v", err)
	}
	got, err := store.FindTask(w, fmt.Sprintf("%03d", tk.Seq))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusDone {
		t.Fatalf("--allow-unverified did not close the task, status=%s", got.Status)
	}
	logSec, _ := got.Doc.Section("Log")
	if !strings.Contains(logSec.Content, "UNVERIFIED") {
		t.Fatalf("an unverified accept must record UNVERIFIED on the task Log; log=%q", logSec.Content)
	}
}

// hasPendingProposal reports whether the task still carries an unconsumed
// box-check proposal — the durability invariant this task defends.
func hasPendingProposal(t *testing.T, w *workspace.Workspace, tk *store.Task) bool {
	t.Helper()
	events, err := eventlog.List(w, eventlog.Query{About: tk.ID, Kinds: []model.EventKind{model.EventComment}, Pending: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if isProposal(e) {
			return true
		}
	}
	return false
}

// TestProposalStaysPendingWhenCloseFails is the dacli 210 regression: a proposal
// must be marked applied ONLY after the task close is durable. Here CloseTask is
// forced to fail (a regular file blocks the done/ status dir MoveTask needs), so
// the accept returns an error and the task never moves to done. If the proposal
// was consumed before that close, the next accept can no longer re-find the task
// and the completed work is permanently invisible — the failure this defends.
func TestProposalStaysPendingWhenCloseFails(t *testing.T) {
	w, tk, ctx := acceptEnv(t)
	deadChild := &agentid.Identity{ID: "a-deadchild", Grant: model.GrantRW, Role: "worker"}
	if err := propose(ctx, w, deadChild, tk); err != nil {
		t.Fatal(err)
	}
	if !hasPendingProposal(t, w, tk) {
		t.Fatal("setup: expected a pending proposal before the accept")
	}

	// Sabotage the close: a regular file where the done/ status directory must
	// be created makes CloseTask's MoveTask fail AFTER proposals would be
	// consumed under the buggy ordering.
	doneDir := w.TasksDir(tk.Project, model.StatusDone)
	if err := os.MkdirAll(filepath.Dir(doneDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doneDir, []byte("blocker"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}
	if err := acceptOne(ctx, w, root, tk, "", false, false, false); err == nil {
		t.Fatal("expected accept to fail while the done/ dir is blocked")
	}
	if tk.Status == model.StatusDone {
		t.Fatal("task must not be marked done when CloseTask failed")
	}
	if !hasPendingProposal(t, w, tk) {
		t.Fatal("proposal was consumed before the close became durable — completed work is now orphaned (dacli 210)")
	}
}

// TestAcceptAllForceReconcilesOrphanedTask covers the `ship` path: a wave's
// spawned agent proposed its own close, then finished and will never sync to
// apply it. `accept --all` alone must still skip it (a live peer might yet own
// it); `accept --all --force` (root only) must adopt and close it, exactly
// like the single-ref override.
func TestAcceptAllForceReconcilesOrphanedTask(t *testing.T) {
	w, tk, ctx := acceptEnv(t)
	deadChild := &agentid.Identity{ID: "a-deadchild", Grant: model.GrantRW, Role: "worker"}
	if err := propose(ctx, w, deadChild, tk); err != nil {
		t.Fatal(err)
	}
	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}

	if err := acceptAll(ctx, w, root, "", false, false, false, false); err != nil {
		t.Fatal(err)
	}
	ref := fmt.Sprintf("%03d", tk.Seq)
	got, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Fatal("accept --all without --force must not close another agent's task")
	}

	if err := acceptAll(ctx, w, root, "", true, false, false, false); err != nil {
		t.Fatal(err)
	}
	got, err = store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusDone {
		t.Fatalf("accept --all --force must close the orphaned task, status=%s", got.Status)
	}
	if got.Owner() != agentid.RootID {
		t.Fatalf("accept --all --force must adopt ownership to root, owner=%s", got.Owner())
	}
}
