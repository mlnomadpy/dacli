package insight

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// doctorEnv builds a workspace holding one task owned by a *different* agent
// that never ran (no proc.txt was ever recorded for it) — the stand-in for a
// spawned child that has since finished and will never sync or accept.
func doctorEnv(t *testing.T) (*workspace.Workspace, *clikit.Ctx) {
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
	return w, &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
}

func TestDoctorFlagsOrphanedTask(t *testing.T) {
	w, ctx := doctorEnv(t)
	tk, err := store.CreateTask(w, "a-deadchild", "p", "Orphaned work", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "orphaned-task") {
		t.Fatalf("expected orphaned-task finding, got:\n%s", out)
	}
	if !strings.Contains(out, "accept --force") {
		t.Fatalf("expected the accept --force suggestion, got:\n%s", out)
	}
	if !strings.Contains(out, tk.Slug) {
		t.Fatalf("expected the orphaned task to be named, got:\n%s", out)
	}
}

func TestDoctorSkipsRootOwnedTask(t *testing.T) {
	w, ctx := doctorEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Root's own work", store.TaskOpts{Accept: []string{"done"}}); err != nil {
		t.Fatal(err)
	}
	if err := cmdDoctor(ctx, nil); err != nil {
		t.Fatal(err)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	if strings.Contains(out, "orphaned-task") {
		t.Fatalf("root-owned task must never be flagged as orphaned, got:\n%s", out)
	}
}
