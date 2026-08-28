package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func reviewOutputFixture(t *testing.T) (*workspace.Workspace, *store.Task, team.Role, string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "x@x"}, {"config", "user.name", "x"}, {"checkout", "-q", "-b", "main"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	w, err := workspace.Init(dir, "review-output")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "p", "Review output", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	branch := fmt.Sprintf("dacli/%03d-%s", task.Seq, task.Slug)
	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", "base"}, {"branch", branch}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit, _ := gitx.Run(dir, "rev-parse", branch)
	tree, _ := gitx.Run(dir, "rev-parse", strings.TrimSpace(commit)+"^{tree}")
	return w, task, team.Role{Name: "reviewer", Grant: "ro", Runtime: "codex", Model: "gpt"}, strings.TrimSpace(commit), strings.TrimSpace(tree)
}

func TestMaterializeReviewOutputLetsROSandboxReturnStructuredResult(t *testing.T) {
	w, task, role, commit, tree := reviewOutputFixture(t)
	result := store.IndependentReviewResult{Schema: store.ReviewResultSchema, Verdict: store.ReviewApprove, ReviewerID: "a-reviewer", ReviewerRole: "reviewer", Runtime: "codex", Model: "gpt", Grant: "ro", IndependentOf: []string{"a-builder"}, CommitSHA: commit, TreeSHA: tree, ObservedAt: time.Now()}
	raw, _ := json.Marshal(result)
	transcript := filepath.Join(t.TempDir(), "transcript.log")
	if err := os.WriteFile(transcript, []byte("ordinary provider output\n"+store.ReviewOutputMarker+string(raw)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := materializeReviewOutput(w, task, "a-reviewer", role, "codex", "gpt", transcript); err != nil {
		t.Fatal(err)
	}
	events, err := eventlog.List(w, eventlog.Query{Actor: "a-reviewer", About: task.ID, Kinds: []model.EventKind{model.EventReview}})
	if err != nil || len(events) != 1 {
		t.Fatalf("materialized events=%d err=%v", len(events), err)
	}
}

func TestMaterializeReviewOutputRefusesMissingAndStaleOutput(t *testing.T) {
	w, task, role, _, _ := reviewOutputFixture(t)
	transcript := filepath.Join(t.TempDir(), "transcript.log")
	if err := os.WriteFile(transcript, []byte("review complete\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := materializeReviewOutput(w, task, "a-reviewer", role, "codex", "gpt", transcript); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing output=%v, want refusal", err)
	}
	stale := store.IndependentReviewResult{Schema: store.ReviewResultSchema, Verdict: store.ReviewApprove, ReviewerID: "a-reviewer", ReviewerRole: "reviewer", Runtime: "codex", Model: "gpt", Grant: "ro", IndependentOf: []string{"a-builder"}, CommitSHA: "stale", TreeSHA: "stale", ObservedAt: time.Now()}
	raw, _ := json.Marshal(stale)
	if err := os.WriteFile(transcript, []byte(store.ReviewOutputMarker+string(raw)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := materializeReviewOutput(w, task, "a-reviewer", role, "codex", "gpt", transcript); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale output=%v, want refusal", err)
	}
}

func TestGenericReviewMayUseLegacyOutputButGovernedReviewRequiresEnvelope(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Review compatibility", store.TaskOpts{})
	if _, err := gitx.Run(w.Root, "branch", taskBranch(task)); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "quiet-reviewer")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho review completed\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustRuntime(t, w, store.Runtime{Name: "quiet-reviewer", Binary: bin, Mode: "stdin", SandboxRO: []string{"--ro"}})

	ctx, _, _ := newCtx(w.Root)
	legacy := []string{"--task", task.ID, "--runtime", "quiet-reviewer", "--grant", "ro", "--review", "--cooperative"}
	if err := cmdSpawn(ctx, legacy); err != nil {
		t.Fatalf("generic review lost backward compatibility: %v", err)
	}
	governed := append(append([]string{}, legacy...), "--structured-review-result")
	if err := cmdSpawn(ctx, governed); err == nil || !strings.Contains(err.Error(), "structured review output") || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("governed review without envelope=%v, want missing structured result failure", err)
	}
}
