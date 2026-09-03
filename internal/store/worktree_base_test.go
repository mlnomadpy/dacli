package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func worktreeBaseGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func worktreeBaseCommit(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	worktreeBaseGit(t, dir, "add", name)
	worktreeBaseGit(t, dir, "commit", "-q", "-m", name)
	return worktreeBaseGit(t, dir, "rev-parse", "HEAD")
}

func TestResolveTaskWorktreeBaseFreshensConfiguredRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	worktreeBaseGit(t, t.TempDir(), "init", "--bare", "-q", remote)

	root := t.TempDir()
	worktreeBaseGit(t, root, "init", "-q", "-b", "main")
	worktreeBaseGit(t, root, "config", "user.email", "test@example.com")
	worktreeBaseGit(t, root, "config", "user.name", "Test")
	worktreeBaseCommit(t, root, "base.txt")
	worktreeBaseGit(t, root, "remote", "add", "origin", remote)
	worktreeBaseGit(t, root, "push", "-q", "-u", "origin", "main")
	worktreeBaseGit(t, root, "checkout", "-q", "-b", "release")
	worktreeBaseCommit(t, root, "release-one.txt")
	worktreeBaseGit(t, root, "push", "-q", "-u", "origin", "release")

	other := t.TempDir()
	worktreeBaseGit(t, other, "clone", "-q", remote, ".")
	worktreeBaseGit(t, other, "config", "user.email", "test@example.com")
	worktreeBaseGit(t, other, "config", "user.name", "Test")
	worktreeBaseGit(t, other, "checkout", "-q", "release")
	want := worktreeBaseCommit(t, other, "release-two.txt")
	worktreeBaseGit(t, other, "push", "-q", "origin", "release")

	worktreeBaseGit(t, root, "checkout", "-q", "main")
	worktreeBaseGit(t, root, "checkout", "-q", "-b", "operator-feature")
	operatorHead := worktreeBaseCommit(t, root, "unrelated.txt")

	w, err := workspace.Init(root, "worktree-base")
	if err != nil {
		t.Fatal(err)
	}
	project, err := CreateProject(w, "a-root", "Project", "p", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ConfigureProjectLanding(project, model.LandingPolicy{Mode: model.LandingPR, Base: "release"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveProject(project); err != nil {
		t.Fatal(err)
	}
	task, err := CreateTask(w, "a-root", "p", "Exact base", TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}

	got, err := ResolveTaskWorktreeBase(w, task)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != TaskWorktreeBaseSchema || got.Branch != "release" || got.Ref != "refs/remotes/origin/release" || got.Commit != want || got.Source != "fresh-origin" || !got.Configured {
		t.Fatalf("base = %#v, want release at %s", got, want)
	}
	if got.Commit == operatorHead {
		t.Fatalf("base inherited operator feature HEAD %s", operatorHead)
	}
}

func TestResolveTaskWorktreeBaseRefusesMissingConfiguredRemoteBase(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	worktreeBaseGit(t, t.TempDir(), "init", "--bare", "-q", remote)
	root := t.TempDir()
	worktreeBaseGit(t, root, "init", "-q", "-b", "main")
	worktreeBaseGit(t, root, "config", "user.email", "test@example.com")
	worktreeBaseGit(t, root, "config", "user.name", "Test")
	worktreeBaseCommit(t, root, "base.txt")
	worktreeBaseGit(t, root, "remote", "add", "origin", remote)
	worktreeBaseGit(t, root, "push", "-q", "-u", "origin", "main")

	w, err := workspace.Init(root, "missing-base")
	if err != nil {
		t.Fatal(err)
	}
	project, err := CreateProject(w, "a-root", "Project", "p", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ConfigureProjectLanding(project, model.LandingPolicy{Mode: model.LandingPR, Base: "release"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveProject(project); err != nil {
		t.Fatal(err)
	}
	task, err := CreateTask(w, "a-root", "p", "Missing base", TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = ResolveTaskWorktreeBase(w, task)
	if err == nil || !strings.Contains(err.Error(), "origin/release") {
		t.Fatalf("missing configured remote base = %v", err)
	}
	if got := worktreeBaseGit(t, root, "branch", "--list", "dacli/001-missing-base"); got != "" {
		t.Fatal("base observation failure created a task branch")
	}
}
