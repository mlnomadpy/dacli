package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/store"
)

func initExecGitRepo(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"add", "."},
		{"commit", "-qm", "initial"},
	} {
		if out, err := gitx.Run(root, args...); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}

func TestResolveSpawnWorkDirResumesSameTaskWorktreeWithoutFlag(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Resume here", store.TaskOpts{})
	branch := taskBranch(task)
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if _, err := gitx.AddWorktree(w.Root, wt, branch, "main"); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(wt, "internal", "features")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, isolated, err := resolveSpawnWorkDir(w, task, nested, false)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedPath(got) != resolvedPath(wt) || !isolated {
		t.Fatalf("resolved (%q, %v), want same-task worktree (%q, true)", got, isolated, wt)
	}
	if head, err := gitx.Run(got, "branch", "--show-current"); err != nil || strings.TrimSpace(head) != branch {
		t.Fatalf("resolved checkout branch = %q, err %v; want %q", strings.TrimSpace(head), err, branch)
	}
}

func TestValidateReviewTargetRefusesMissingLocalTaskBranch(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Missing review branch", store.TaskOpts{})
	f, err := clikit.ParseFlags([]string{"--review"})
	if err != nil {
		t.Fatal(err)
	}

	err = validateReviewTarget(w, task, f)
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("exit = %d, want policy refusal 3: %v", clikit.ExitCode(err), err)
	}
	for _, want := range []string{task.ID, taskBranch(task), "create or restore it"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal missing %q: %v", want, err)
		}
	}
}

func TestSpawnFromTaskWorktreeRunsAndEditsOnlyThere(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Resume runtime", store.TaskOpts{})
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if _, err := gitx.AddWorktree(w.Root, wt, taskBranch(task), "main"); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "record-cwd")
	script := "#!/bin/sh\npwd > runtime-cwd.txt\ngit branch --show-current > runtime-branch.txt\nprintf resumed > resumed-edit.txt\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	mustRuntime(t, w, store.Runtime{Name: "cwd-recorder", Binary: bin, Mode: "stdin"})

	ctx, _, _ := newCtx(wt)
	if err := cmdSpawn(ctx, []string{"--task", task.ID, "--runtime", "cwd-recorder", "--grant", "rw", "--cooperative"}); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"runtime-cwd.txt":    resolvedPath(wt),
		"runtime-branch.txt": taskBranch(task),
		"resumed-edit.txt":   "resumed",
	} {
		raw, err := os.ReadFile(filepath.Join(wt, name))
		if err != nil {
			t.Fatalf("worktree %s: %v", name, err)
		}
		got := strings.TrimSpace(string(raw))
		if name == "runtime-cwd.txt" {
			got = resolvedPath(got)
		}
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
		if _, err := os.Stat(filepath.Join(w.Root, name)); !os.IsNotExist(err) {
			t.Errorf("%s escaped into main checkout (stat err %v)", name, err)
		}
	}
}

func TestResolveSpawnWorkDirKeepsMainCheckoutBehavior(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Main stays main", store.TaskOpts{})
	other := mustTask(t, w, "Unrelated", store.TaskOpts{})
	otherWT := w.WorktreePath(other.Project, other.Seq, other.Slug)
	if _, err := gitx.AddWorktree(w.Root, otherWT, taskBranch(other), "main"); err != nil {
		t.Fatal(err)
	}

	got, isolated, err := resolveSpawnWorkDir(w, task, w.Root, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != w.Root || isolated {
		t.Fatalf("resolved (%q, %v), want main checkout (%q, false)", got, isolated, w.Root)
	}
}

func TestResolveSpawnWorkDirRefusesMismatchedTaskWorktree(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Wanted", store.TaskOpts{})
	other := mustTask(t, w, "Wrong checkout", store.TaskOpts{})
	otherWT := w.WorktreePath(other.Project, other.Seq, other.Slug)
	if _, err := gitx.AddWorktree(w.Root, otherWT, taskBranch(other), "main"); err != nil {
		t.Fatal(err)
	}

	_, _, err := resolveSpawnWorkDir(w, task, otherWT, false)
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("exit = %d, want policy refusal 3: %v", clikit.ExitCode(err), err)
	}
	for _, want := range []string{taskBranch(other), "dacli spawn --task " + task.ID + " --worktree"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not name %q: %v", want, err)
		}
	}
}

func TestResolveSpawnWorkDirRefusesAmbiguousTaskWorktrees(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "Ambiguous", store.TaskOpts{})
	branch := taskBranch(task)
	wts := []gitx.Worktree{{Path: "/tmp/a", Branch: branch}, {Path: "/tmp/b", Branch: branch}}

	_, _, err := resolveSpawnWorkDirFrom(w, task, "/tmp/a", false, wts)
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("exit = %d, want policy refusal 3: %v", clikit.ExitCode(err), err)
	}
	if want := "git worktree remove <duplicate-path>"; !strings.Contains(err.Error(), want) {
		t.Fatalf("refusal does not name recovery command %q: %v", want, err)
	}
}
