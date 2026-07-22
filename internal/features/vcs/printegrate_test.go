package vcs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// prIntegrateEnv builds a real git repo on main with a workspace holding one
// DONE task whose branch (dacli/001-slug) carries a commit ahead of main, ready
// to integrate. DACLI_AGENT is cleared so the actor is root (rw).
func prIntegrateEnv(t *testing.T) (string, *workspace.Workspace, *store.Task) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("DACLI_AGENT", "")
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-q", "-m", "base")

	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	// A real task branch with a commit ahead of main so a local-merge fallback
	// has something to merge.
	branch := BranchFor(tk)
	gitAt(t, dir, "checkout", "-q", "-b", branch)
	if err := os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "feature.txt")
	gitAt(t, dir, "commit", "-q", "-m", "feature work")
	gitAt(t, dir, "checkout", "-q", "main")
	return dir, w, tk
}

func prCtx(dir string) (*clikit.Ctx, *bytes.Buffer) {
	var out bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &out, Cwd: dir}, &out
}

// stubGH captures gh invocations and returns canned output, so the PR path is
// exercised without a live GitHub. It restores the real runner on cleanup.
func stubGH(t *testing.T, fn func(dir string, args ...string) (string, error)) *[][]string {
	t.Helper()
	var calls [][]string
	orig := runGH
	runGH = func(dir string, args ...string) (string, error) {
		calls = append(calls, args)
		return fn(dir, args...)
	}
	t.Cleanup(func() { runGH = orig })
	return &calls
}

func stubPush(t *testing.T, fn func(root, branch string) (string, error)) *[][]string {
	t.Helper()
	var calls [][]string
	orig := pushBranch
	pushBranch = func(root, branch string) (string, error) {
		calls = append(calls, []string{root, branch})
		return fn(root, branch)
	}
	t.Cleanup(func() { pushBranch = orig })
	return &calls
}

// --pr pushes the branch, opens a PR (recorded), and merges via gh pr merge.
func TestIntegratePRPushesOpensAndMerges(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	push := stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/7", nil
		}
		return "merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr: %v\n%s", err, out.String())
	}

	if len(*push) != 1 {
		t.Fatalf("expected one push, got %v", *push)
	}
	var sawCreate, sawMerge bool
	for _, c := range *gh {
		joined := strings.Join(c, " ")
		if strings.HasPrefix(joined, "pr create") {
			sawCreate = true
		}
		if strings.HasPrefix(joined, "pr merge") {
			sawMerge = true
			if !strings.Contains(joined, "--squash") || !strings.Contains(joined, "--delete-branch") {
				t.Errorf("gh pr merge missing --squash/--delete-branch: %q", joined)
			}
		}
	}
	if !sawCreate || !sawMerge {
		t.Fatalf("expected gh pr create and pr merge, got %v", *gh)
	}
	// The PR URL was recorded as a comment event on the task.
	events, _ := eventlog.List(w, eventlog.Query{About: tk.ID, Kinds: []model.EventKind{model.EventComment}})
	found := false
	for _, e := range events {
		if strings.Contains(e.Body, "PR opened: https://github.com/acme/widgets/pull/7") {
			found = true
		}
	}
	if !found {
		t.Errorf("PR URL not recorded as a comment event")
	}
	if !strings.Contains(out.String(), "merged via gh") {
		t.Errorf("expected a merged-via-gh notice:\n%s", out.String())
	}
}

// --no-merge opens the PR and STOPS: gh pr merge is never called.
func TestIntegratePRNoMergeStopsBeforeMerge(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/9", nil
		}
		return "", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--no-merge", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr --no-merge: %v\n%s", err, out.String())
	}
	for _, c := range *gh {
		if len(c) >= 2 && c[0] == "pr" && c[1] == "merge" {
			t.Errorf("--no-merge still called gh pr merge: %v", c)
		}
	}
	if !strings.Contains(out.String(), "left open for human review") {
		t.Errorf("expected a human-review notice:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "none merged (--no-merge)") {
		t.Errorf("expected a none-merged summary:\n%s", out.String())
	}
	// main did not advance (nothing merged): feature.txt is absent.
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(err) {
		t.Errorf("--no-merge merged the branch into main (feature.txt present)")
	}
}

// A network failure at push falls back to a LOCAL merge with a warning, so the
// wave still lands offline. gh is never reached.
func TestIntegratePRFallsBackToLocalMergeOnPushNetworkError(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) {
		return "fatal: unable to access 'https://github.com/...': Could not resolve host: github.com", fmt.Errorf("exit status 128")
	})
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		t.Errorf("gh must not be called after a push network failure: %v", args)
		return "", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr (fallback): %v\n%s", err, out.String())
	}
	if len(*gh) != 0 {
		t.Errorf("gh was called despite the push network failure: %v", *gh)
	}
	if !strings.Contains(out.String(), "falling back to a local merge") {
		t.Errorf("expected a fallback warning:\n%s", out.String())
	}
	// The local merge landed: feature.txt is now on main.
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); err != nil {
		t.Errorf("local-merge fallback did not land the branch: %v", err)
	}
}

// A NON-network push failure (e.g. a protected branch) is surfaced, NOT
// silently local-merged.
func TestIntegratePRSurfacesNonNetworkPushError(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) {
		return "remote: error: GH006: Protected branch update failed", fmt.Errorf("exit status 1")
	})
	stubGH(t, func(dir string, args ...string) (string, error) { return "", nil })

	ctx, out := prCtx(dir)
	err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"})
	if err == nil {
		t.Fatalf("expected a hard error for a non-network push failure\n%s", out.String())
	}
	// It did NOT fall back to a local merge.
	if _, statErr := os.Stat(filepath.Join(dir, "feature.txt")); statErr == nil {
		t.Errorf("a non-network push failure was silently local-merged")
	}
}

// --no-merge does NOT fall back to a local merge when offline: the operator
// asked for a PR, so an offline failure is surfaced rather than merged behind
// their back.
func TestIntegratePRNoMergeDoesNotFallBackOffline(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) {
		return "Could not resolve host: github.com", fmt.Errorf("exit status 128")
	})
	stubGH(t, func(dir string, args ...string) (string, error) { return "", nil })

	ctx, out := prCtx(dir)
	err := cmdIntegrate(ctx, []string{"--pr", "--no-merge", "--tasks", tk.ID, "--into", "main"})
	if err == nil {
		t.Fatalf("expected an error: --no-merge offline must not local-merge\n%s", out.String())
	}
	if _, statErr := os.Stat(filepath.Join(dir, "feature.txt")); statErr == nil {
		t.Errorf("--no-merge fell back to a local merge while offline")
	}
}
