package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func parentCommitFixture(t *testing.T, claims []string) (*workspace.Workspace, *Task, RootHandoff) {
	t.Helper()
	w := handoffRepo(t)
	if _, err := CreateProject(w, "a-root", "Parent commit", "parent-commit", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := CreateTask(w, "a-root", "parent-commit", "Change claimed source", TaskOpts{Accept: []string{"source.txt is changed"}, Claims: claims})
	if err != nil {
		t.Fatal(err)
	}
	worktree := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if err := os.MkdirAll(filepath.Dir(worktree), 0o755); err != nil {
		t.Fatal(err)
	}
	branch := "dacli/001-" + task.Slug
	if _, err := gitx.Run(w.Root, "worktree", "add", "-b", branch, worktree, "HEAD"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = gitx.Run(w.Root, "worktree", "remove", "--force", worktree) })
	if err := os.WriteFile(filepath.Join(worktree, "source.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runID := "01PARENTCOMMITREQUEST0001"
	if err := os.MkdirAll(w.RunDir(runID), 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: runID, Task: task.ID, Child: "a-worker", Role: "fixer", Claims: claims, Started: time.Unix(10, 0)}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	h, ok, err := CaptureRootHandoff(w, runID, task.ID, rec.Child, worktree, RootHandoffRequest{
		Schema: RootHandoffSchema, CommitMessage: "Change claimed source",
		Verification:    []RootHandoffVerification{{Command: "test source", ExitCode: 0, Result: "pass"}},
		FailedOperation: "git index lock", FailureClass: "filesystem_sandbox_refusal", NextAction: "parent creates exact commit",
	}, time.Unix(20, 0))
	if err != nil || !ok {
		t.Fatalf("capture handoff ok=%t err=%v", ok, err)
	}
	return w, task, h
}

func TestParentCommitRequestIsExactClaimBoundAndIdempotent(t *testing.T) {
	w, task, handoff := parentCommitFixture(t, []string{"source.txt"})
	receipt, err := ApplyParentCommit(w, handoff, time.Unix(30, 0))
	if err != nil {
		t.Fatal(err)
	}
	request, err := LoadParentCommitRequest(w, handoff.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if request.Schema != ParentCommitRequestSchema || request.TaskID != task.ID || request.RunID != handoff.RunID || request.ChildID != "a-worker" || request.Branch != "dacli/001-"+task.Slug || request.ParentCommit == "" || request.TreeOID == "" || request.DiffSHA256 != handoff.DiffSHA256 || request.TreeSHA256 != handoff.TreeSHA256 || len(request.ChangedPaths) != 1 || len(request.Verification) != 1 {
		t.Fatalf("incomplete exact request: %+v", request)
	}
	if receipt.Schema != ParentCommitReceiptSchema || receipt.RequestID != request.RequestID || receipt.Commit == "" || receipt.TreeOID != request.TreeOID {
		t.Fatalf("receipt=%+v request=%+v", receipt, request)
	}
	if _, err := os.Stat(filepath.Join(w.RunDir(handoff.RunID), RootHandoffConsumedFile)); err != nil {
		t.Fatalf("parent commit did not write handoff consumption audit: %v", err)
	}
	message, err := gitx.Run(handoff.Worktree, "show", "-s", "--format=%B", receipt.Commit)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Change claimed source", "Dacli-Agent: a-worker", "Dacli-Role: fixer", "Dacli-Task: 001-" + task.Slug} {
		if !strings.Contains(message, want) {
			t.Fatalf("commit message missing %q:\n%s", want, message)
		}
	}
	if dirty, err := gitx.DirtyPaths(handoff.Worktree, workspace.Dir); err != nil || len(dirty) != 0 {
		t.Fatalf("parent commit left dirty paths=%v err=%v", dirty, err)
	}
	if err := os.Remove(ParentCommitReceiptPath(w, handoff.RunID)); err != nil {
		t.Fatal(err)
	}
	again, err := ApplyParentCommit(w, handoff, time.Unix(40, 0))
	if err != nil || again.Commit != receipt.Commit || again.RequestID != receipt.RequestID {
		t.Fatalf("idempotent apply=%+v err=%v, want %+v", again, err, receipt)
	}
}

func TestParentCommitRefusesExtraOrStaleWorkerWrites(t *testing.T) {
	t.Run("outside-claim", func(t *testing.T) {
		w, _, handoff := parentCommitFixture(t, []string{"source.txt"})
		if err := os.WriteFile(filepath.Join(handoff.Worktree, "extra.txt"), []byte("outside\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ApplyParentCommit(w, handoff, time.Unix(30, 0)); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("extra write apply=%v, want stale refusal", err)
		}
	})

	t.Run("claim-does-not-cover-observed-path", func(t *testing.T) {
		w, _, handoff := parentCommitFixture(t, []string{"docs"})
		if _, err := ApplyParentCommit(w, handoff, time.Unix(30, 0)); err == nil || !strings.Contains(err.Error(), "outside claims") {
			t.Fatalf("outside-claim apply=%v, want claim refusal", err)
		}
	})
}
