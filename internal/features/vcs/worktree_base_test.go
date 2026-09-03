package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestWorktreeAddStartsFromConfiguredBaseNotOperatorHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	unsetAgentEnv(t)
	remoteParent := t.TempDir()
	remote := filepath.Join(remoteParent, "origin.git")
	gitAt(t, remoteParent, "init", "--bare", "-q", remote)
	root := t.TempDir()
	gitAt(t, root, "init", "-q", "-b", "main")
	gitAt(t, root, "config", "user.email", "test@example.com")
	gitAt(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, root, "add", "base.txt")
	gitAt(t, root, "commit", "-q", "-m", "base")
	base := gitAt(t, root, "rev-parse", "HEAD")
	gitAt(t, root, "remote", "add", "origin", remote)
	gitAt(t, root, "push", "-q", "-u", "origin", "main")

	w, err := workspace.Init(root, "worktree-base")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(w, "a-root", "Project", "p", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ConfigureProjectLanding(project, model.LandingPolicy{Mode: model.LandingPR, Base: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "p", "Exact base", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}

	gitAt(t, root, "checkout", "-q", "-b", "operator-feature")
	if err := os.WriteFile(filepath.Join(root, "unrelated.txt"), []byte("unrelated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, root, "add", "unrelated.txt")
	gitAt(t, root, "commit", "-q", "-m", "operator feature")

	ctx, out := prCtx(root)
	if err := cmdWorktreeAdd(ctx, []string{"--task", task.ID}); err != nil {
		t.Fatal(err)
	}
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if got := gitAt(t, wt, "rev-parse", "HEAD"); got != base {
		t.Fatalf("worktree HEAD = %s, want configured base %s", got, base)
	}
	if _, err := os.Stat(filepath.Join(wt, "unrelated.txt")); !os.IsNotExist(err) {
		t.Fatalf("worktree inherited operator feature: %v", err)
	}
	for _, want := range []string{"worktree base: main at " + base, "fresh-origin"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}
