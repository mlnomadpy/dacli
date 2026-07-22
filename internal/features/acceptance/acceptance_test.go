package acceptance

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
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
	t.Setenv("DACLI_AGENT", "") // act as root
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
