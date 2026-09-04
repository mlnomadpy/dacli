package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func claimSandboxFixture(t *testing.T, claims []string) (*claimSandbox, string) {
	t.Helper()
	w := newExecWS(t)
	if err := os.WriteFile(filepath.Join(w.Root, "claimed.txt"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	initExecGitRepo(t, w.Root)
	runID := "01CLAIMSANDBOX0000000001"
	if err := os.MkdirAll(w.RunDir(runID), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := prepareClaimSandbox(w, runID, w.Root, claims, time.Unix(10, 0))
	if err != nil {
		t.Fatal(err)
	}
	return &plan, w.Root
}

func TestClaimSandboxFirstOutOfClaimWriteNeverMutatesCanonicalCheckout(t *testing.T) {
	plan, root := claimSandboxFixture(t, []string{"claimed.txt"})
	if err := os.WriteFile(filepath.Join(plan.SandboxDir, "rogue.txt"), []byte("must stay disposable\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := workspaceFind(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = projectClaimSandbox(w, plan.RunID, time.Unix(20, 0))
	if err == nil || !strings.Contains(err.Error(), "outside exact claims") || !strings.Contains(err.Error(), "canonical checkout was not mutated") {
		t.Fatalf("out-of-claim projection = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "rogue.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("first out-of-claim write reached canonical checkout: %v", statErr)
	}
}

func TestClaimSandboxProjectsOnlyClaimedRegularFilesAndIsIdempotent(t *testing.T) {
	plan, root := claimSandboxFixture(t, []string{"claimed.txt", "new/allowed.txt"})
	if remotes, err := gitx.Run(plan.SandboxDir, "remote"); err != nil || strings.TrimSpace(remotes) != "" {
		t.Fatalf("claim sandbox retained canonical remote %q err=%v", remotes, err)
	}
	if err := os.WriteFile(filepath.Join(plan.SandboxDir, "claimed.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(plan.SandboxDir, "new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.SandboxDir, "new", "allowed.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := workspaceFind(root)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := projectClaimSandbox(w, plan.RunID, time.Unix(20, 0))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "claimed.txt,new/allowed.txt" {
		t.Fatalf("projected paths = %v", paths)
	}
	paths, err = projectClaimSandbox(w, plan.RunID, time.Unix(30, 0))
	if err != nil || strings.Join(paths, ",") != "claimed.txt,new/allowed.txt" {
		t.Fatalf("idempotent projection paths=%v err=%v", paths, err)
	}
	for path, want := range map[string]string{"claimed.txt": "after", "new/allowed.txt": "new"} {
		raw, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil || strings.TrimSpace(string(raw)) != want {
			t.Fatalf("%s = %q err=%v", path, raw, readErr)
		}
	}
}

func TestClaimSandboxFailsClosedOnTraversalDeleteSymlinkAndMultiRootEscape(t *testing.T) {
	t.Run("parent traversal", func(t *testing.T) {
		w := newExecWS(t)
		initExecGitRepo(t, w.Root)
		if err := os.MkdirAll(w.RunDir("01TRAVERSAL00000000000001"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := prepareClaimSandbox(w, "01TRAVERSAL00000000000001", w.Root, []string{"../outside"}, time.Now()); err == nil || !strings.Contains(err.Error(), "repository-relative") {
			t.Fatalf("traversal claim = %v", err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		plan, root := claimSandboxFixture(t, []string{"claimed.txt"})
		if err := os.Remove(filepath.Join(plan.SandboxDir, "claimed.txt")); err != nil {
			t.Fatal(err)
		}
		w, _ := workspaceFind(root)
		if _, err := projectClaimSandbox(w, plan.RunID, time.Now()); err == nil || !strings.Contains(err.Error(), "rename, copy, or delete") {
			t.Fatalf("delete projection = %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "claimed.txt")); err != nil {
			t.Fatalf("canonical file was deleted: %v", err)
		}
	})
	t.Run("introduced symlink", func(t *testing.T) {
		plan, root := claimSandboxFixture(t, []string{"new-link"})
		if err := os.Symlink("claimed.txt", filepath.Join(plan.SandboxDir, "new-link")); err != nil {
			t.Fatal(err)
		}
		w, _ := workspaceFind(root)
		if _, err := projectClaimSandbox(w, plan.RunID, time.Now()); err == nil || !strings.Contains(err.Error(), "non-regular") {
			t.Fatalf("symlink projection = %v", err)
		}
		if _, err := os.Lstat(filepath.Join(root, "new-link")); !os.IsNotExist(err) {
			t.Fatalf("canonical symlink appeared: %v", err)
		}
	})
	t.Run("multi-root is all-or-nothing", func(t *testing.T) {
		plan, root := claimSandboxFixture(t, []string{"claimed.txt", "other/allowed.txt"})
		if err := os.WriteFile(filepath.Join(plan.SandboxDir, "claimed.txt"), []byte("would be allowed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(plan.SandboxDir, "rogue.txt"), []byte("escape\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		w, _ := workspaceFind(root)
		if _, err := projectClaimSandbox(w, plan.RunID, time.Now()); err == nil || !strings.Contains(err.Error(), "outside exact claims") {
			t.Fatalf("multi-root escape projection = %v", err)
		}
		raw, err := os.ReadFile(filepath.Join(root, "claimed.txt"))
		if err != nil || strings.TrimSpace(string(raw)) != "before" {
			t.Fatalf("partial claimed projection occurred: %q err=%v", raw, err)
		}
	})
}

// workspaceFind is a test seam kept tiny so the fixture proves the sandbox's
// path-based shared-workspace redirect too.
var workspaceFind = workspace.Find

func TestClaimSandboxProviderHelper(t *testing.T) {
	if os.Getenv("DACLI_CLAIM_SANDBOX_HELPER") != "1" {
		return
	}
	if err := os.WriteFile("rogue.txt", []byte("first write\n"), 0o644); err != nil {
		os.Exit(41)
	}
}

func TestProviderFirstOutOfClaimWriteCannotMutateTaskWorktree(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Enforce exact claim", store.TaskOpts{Accept: []string{"claimed.txt may change"}, Claims: []string{"claimed.txt"}})
	runtime := store.Runtime{
		Name: "claim-escape-provider", Harness: "generic", Binary: os.Args[0], Mode: "stdin",
		GlobalArgs: []string{"-test.run=TestClaimSandboxProviderHelper", "--"},
		Env:        []string{"DACLI_CLAIM_SANDBOX_HELPER"},
	}
	mustRuntime(t, w, runtime)
	t.Setenv("DACLI_CLAIM_SANDBOX_HELPER", "1")
	ctx, _, _ := newCtx(w.Root)
	err := cmdSpawn(ctx, []string{"--task", task.ID, "--runtime", runtime.Name, "--grant", string(model.GrantRW), "--worktree", "--claim", "claimed.txt"})
	if err == nil || !strings.Contains(err.Error(), "outside exact claims") {
		t.Fatalf("out-of-claim provider spawn = %v", err)
	}
	canonical := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if _, statErr := os.Stat(filepath.Join(canonical, "rogue.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("provider's first out-of-claim write reached task worktree: %v", statErr)
	}
}
