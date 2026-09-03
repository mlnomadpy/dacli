package execution

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
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
	missingRun := "01REVIEWMISSINGRESULT00001"
	if err := os.MkdirAll(w.RunDir(missingRun), 0o755); err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(w.RunDir(missingRun), "transcript.log")
	if err := os.WriteFile(transcript, []byte("review complete\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := materializeReviewOutput(w, task, "a-reviewer", role, "codex", "gpt", transcript); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing output=%v, want refusal", err)
	}
	if !store.RootHandoffRequested(w, missingRun) {
		t.Fatal("missing structured result was not durably distinguished from an empty successful review")
	}
	staleRun := "01REVIEWSTALERESULT000001"
	if err := os.MkdirAll(w.RunDir(staleRun), 0o755); err != nil {
		t.Fatal(err)
	}
	transcript = filepath.Join(w.RunDir(staleRun), "transcript.log")
	stale := store.IndependentReviewResult{Schema: store.ReviewResultSchema, Verdict: store.ReviewApprove, ReviewerID: "a-reviewer", ReviewerRole: "reviewer", Runtime: "codex", Model: "gpt", Grant: "ro", IndependentOf: []string{"a-builder"}, CommitSHA: "stale", TreeSHA: "stale", ObservedAt: time.Now()}
	raw, _ := json.Marshal(stale)
	if err := os.WriteFile(transcript, []byte(store.ReviewOutputMarker+string(raw)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := materializeReviewOutput(w, task, "a-reviewer", role, "codex", "gpt", transcript); err == nil || !strings.Contains(err.Error(), "commit_sha") || !strings.Contains(err.Error(), "tree_sha") {
		t.Fatalf("stale output=%v, want refusal", err)
	}
	if !store.RootHandoffRequested(w, staleRun) {
		t.Fatal("stale structured result was not durably distinguished from an exact-tree review")
	}
}

func TestReviewValidationReportsEveryExpectedAndActualFieldWithoutApproval(t *testing.T) {
	tests := map[string]func(*store.IndependentReviewResult){
		"schema":        func(r *store.IndependentReviewResult) { r.Schema = "independent-review-result/v0" },
		"reviewer_id":   func(r *store.IndependentReviewResult) { r.ReviewerID = "a-other" },
		"reviewer_role": func(r *store.IndependentReviewResult) { r.ReviewerRole = "implementer" },
		"runtime":       func(r *store.IndependentReviewResult) { r.Runtime = "claude" },
		"model":         func(r *store.IndependentReviewResult) { r.Model = "other" },
		"grant":         func(r *store.IndependentReviewResult) { r.Grant = "rw" },
		"commit_sha":    func(r *store.IndependentReviewResult) { r.CommitSHA = "other-commit" },
		"tree_sha":      func(r *store.IndependentReviewResult) { r.TreeSHA = "other-tree" },
	}
	for field, mutate := range tests {
		t.Run(field, func(t *testing.T) {
			w, task, role, commit, tree := reviewOutputFixture(t)
			result := store.IndependentReviewResult{
				Schema: store.ReviewResultSchema, Verdict: store.ReviewApprove,
				ReviewerID: "a-reviewer", ReviewerRole: "reviewer", Runtime: "codex", Model: "gpt", Grant: "ro",
				IndependentOf: []string{"a-builder"}, CommitSHA: commit, TreeSHA: tree, ObservedAt: time.Now(),
			}
			mutate(&result)
			runID := "01REVIEWVALIDATION" + strings.ToUpper(strings.ReplaceAll(field, "_", ""))
			runDir := w.RunDir(runID)
			if err := os.MkdirAll(runDir, 0o755); err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(result)
			transcript := filepath.Join(runDir, "transcript.log")
			if err := os.WriteFile(transcript, []byte(store.ReviewOutputMarker+string(raw)+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			err := materializeReviewOutput(w, task, "a-reviewer", role, "codex", "gpt", transcript)
			var validation *store.ReviewValidationError
			if !errors.As(err, &validation) || !slices.Contains(validation.Diagnostic.Mismatches, field) {
				t.Fatalf("%s mismatch = %v, diagnostic=%+v", field, err, validation)
			}
			if validation.Diagnostic.Schema != store.ReviewValidationSchema || validation.Diagnostic.Expected.Schema == "" || validation.Diagnostic.Actual.Schema == "" || validation.Diagnostic.Expected.ReviewerID == "" || validation.Diagnostic.Expected.ReviewerRole == "" || validation.Diagnostic.Expected.Runtime == "" || validation.Diagnostic.Expected.Model == "" || validation.Diagnostic.Expected.Grant == "" || validation.Diagnostic.Expected.CommitSHA == "" || validation.Diagnostic.Expected.TreeSHA == "" {
				t.Fatalf("incomplete structured diagnostic: %+v", validation.Diagnostic)
			}
			persisted, readErr := os.ReadFile(filepath.Join(runDir, "review-validation.json"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			var decoded store.ReviewValidationDiagnostic
			if json.Unmarshal(persisted, &decoded) != nil || !slices.Contains(decoded.Mismatches, field) {
				t.Fatalf("persisted diagnostic = %s", persisted)
			}
			events, listErr := eventlog.List(w, eventlog.Query{About: task.ID, Kinds: []model.EventKind{model.EventReview}})
			if listErr != nil || len(events) != 0 {
				t.Fatalf("invalid %s review recorded approval events=%d err=%v", field, len(events), listErr)
			}
		})
	}
}

func TestReviewPublicationFailureBecomesDurableHandoff(t *testing.T) {
	w, task, role, commit, tree := reviewOutputFixture(t)
	runID := "01REVIEWRESULTFAILURE00001"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	result := store.IndependentReviewResult{
		Schema: store.ReviewResultSchema, Verdict: store.ReviewRequestChanges,
		ReviewerID: "a-reviewer", ReviewerRole: "reviewer", Runtime: "codex", Model: "gpt", Grant: "ro",
		IndependentOf: []string{"a-builder"}, CommitSHA: commit, TreeSHA: tree, ObservedAt: time.Now(),
		Findings: []store.ReviewFinding{{
			ID: "review.finding-1", Severity: "major", File: "internal/store/review.go", Line: 43,
			Evidence: "typed result could not be durably appended", AffectedInvariant: "silence is never approval",
			SuggestedVerification: "rerun the exact-tree independent review",
		}},
	}
	raw, _ := json.Marshal(result)
	transcript := filepath.Join(runDir, "transcript.log")
	if err := os.WriteFile(transcript, []byte(store.ReviewOutputMarker+string(raw)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldAppend := appendReviewResultEvent
	appendReviewResultEvent = func(*workspace.Workspace, string, model.EventKind, string, string, string) (*eventlog.Event, error) {
		return nil, os.ErrPermission
	}
	t.Cleanup(func() { appendReviewResultEvent = oldAppend })

	err := materializeReviewOutput(w, task, "a-reviewer", role, "codex", "gpt", transcript)
	if err == nil || !strings.Contains(err.Error(), "handoff-required") || !store.RootHandoffRequested(w, runID) {
		t.Fatalf("review publication failure = %v, requested=%t", err, store.RootHandoffRequested(w, runID))
	}
	rec := procmon.Record{RunID: runID, Task: task.ID, Child: "a-reviewer", Started: time.Now().Add(-time.Second)}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	summary, err := finalizeRunChecked(w, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "handoff-required") || strings.Contains(summary, "no visible result") {
		t.Fatalf("review finalization summary = %q", summary)
	}
	handoff, err := store.LoadRootHandoff(w, runID)
	if err != nil {
		t.Fatal(err)
	}
	if handoff.FailureClass != "review_result_publication_failure" || handoff.FailedOperation != "persist structured independent-review result" {
		t.Fatalf("review handoff = %+v", handoff)
	}
	if len(handoff.Unresolved) != 1 || !strings.Contains(handoff.Unresolved[0], "review.finding-1") || !strings.Contains(handoff.NextAction, "do not inspect raw transcript") {
		t.Fatalf("review handoff recovery evidence = %+v", handoff)
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
	loaded, err := store.LoadRuntime(w, "quiet-reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRuntimeROProbe(w, loaded, bin, store.RuntimeROVerified, "test fixture"); err != nil {
		t.Fatal(err)
	}

	ctx, _, _ := newCtx(w.Root)
	legacy := []string{"--task", task.ID, "--runtime", "quiet-reviewer", "--grant", "ro", "--review", "--cooperative"}
	if err := cmdSpawn(ctx, legacy); err != nil {
		t.Fatalf("generic review lost backward compatibility: %v", err)
	}
	governed := []string{"--task", task.ID, "--runtime", "quiet-reviewer", "--grant", "ro", "--review", "--structured-review-result"}
	if err := cmdSpawn(ctx, governed); clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "handoff-required") || !strings.Contains(err.Error(), "do not inspect raw transcript") {
		t.Fatalf("governed review without envelope=%v, want durable handoff refusal", err)
	}
}
