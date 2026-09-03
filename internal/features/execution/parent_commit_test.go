package execution

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

func TestParentCommitWorkerHelper(t *testing.T) {
	if os.Getenv("DACLI_PARENT_COMMIT_HELPER") != "1" {
		return
	}
	content := os.Getenv("DACLI_PARENT_COMMIT_CONTENT")
	if err := os.WriteFile("claimed.txt", []byte(content+"\n"), 0o644); err != nil {
		os.Exit(31)
	}
}

func TestGitIndexRestrictedImplementationAndCorrectionAreParentCommitted(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Parent commit implementation", store.TaskOpts{Accept: []string{"claimed.txt records the implementation"}, Claims: []string{"claimed.txt"}})
	runtime := store.Runtime{
		Name: "index-restricted-worker", Harness: "codex", Binary: os.Args[0], Mode: "stdin",
		GlobalArgs: []string{"-test.run=TestParentCommitWorkerHelper", "--"},
		Env:        []string{"DACLI_PARENT_COMMIT_HELPER", "DACLI_PARENT_COMMIT_CONTENT"},
	}
	mustRuntime(t, w, runtime)
	mustRole(t, w, team.Role{Name: "index-restricted-builder", Kind: "implementer", Grant: "rw", Runtime: runtime.Name})
	t.Setenv("DACLI_PARENT_COMMIT_HELPER", "1")

	oldProbe := probeMutationGitLock
	probeMutationGitLock = func(string) error { return errors.New("open index.lock: permission denied") }
	t.Cleanup(func() { probeMutationGitLock = oldProbe })

	var commits []string
	for _, content := range []string{"initial implementation", "review correction"} {
		t.Setenv("DACLI_PARENT_COMMIT_CONTENT", content)
		ctx, _, _ := newCtx(w.Root)
		if err := cmdSpawn(ctx, []string{"--task", task.ID, "--role", "index-restricted-builder", "--grant", "rw", "--worktree", "--claim", "claimed.txt", "--detach"}); err != nil {
			t.Fatalf("%s spawn: %v", content, err)
		}
		runID := store.LatestRunID(w)
		if err := cmdWait(ctx, []string{runID, "--interval", "1", "--timeout", "5"}); err != nil {
			t.Fatalf("%s wait: %v", content, err)
		}
		if err := cmdWait(ctx, []string{runID, "--interval", "1", "--timeout", "5"}); err != nil {
			t.Fatalf("%s restart wait: %v", content, err)
		}
		worktree := w.WorktreePath(task.Project, task.Seq, task.Slug)
		commit, err := gitx.Run(worktree, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		commits = append(commits, strings.TrimSpace(commit))
		got, err := os.ReadFile(filepath.Join(worktree, "claimed.txt"))
		if err != nil || strings.TrimSpace(string(got)) != content {
			t.Fatalf("worktree content=%q err=%v, want %q", got, err, content)
		}
		if dirty, err := gitx.DirtyPaths(worktree, ".dacli"); err != nil || len(dirty) != 0 {
			t.Fatalf("parent commit left dirty worktree paths=%v err=%v", dirty, err)
		}
	}
	if commits[0] == commits[1] {
		t.Fatalf("correction did not create a new exact commit: %v", commits)
	}
	events, err := eventlog.List(w, eventlog.Query{About: task.ID, Kinds: []model.EventKind{model.EventCommit}})
	if err != nil || len(events) != 2 {
		t.Fatalf("parent commit events=%d err=%v", len(events), err)
	}
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		t.Fatal(err)
	}
	receipts := 0
	for _, entry := range entries {
		if _, err := os.Stat(store.ParentCommitReceiptPath(w, entry.Name())); err == nil {
			receipts++
		}
	}
	if receipts != 2 {
		t.Fatalf("parent commit receipts=%d, want implementation and correction", receipts)
	}
}
