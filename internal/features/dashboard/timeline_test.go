package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestDeliveryTimelineBindsCurrentAttemptAndRefusesCorruptEvidence(t *testing.T) {
	w := dashboardEnv(t)
	task, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	runID := "01TIMELINECURRENT0000000000"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{RunID: runID, Child: "a-worker", Task: task.ID, Role: "implementer", Runtime: "codex", Started: started, Outcome: "OK"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "usage.txt"), []byte("input_tokens: 10\noutput_tokens: 25\nnum_turns: 3\ncost_usd: 0.125\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	journalDir := filepath.Join(w.Root, workspace.Dir, "loop")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	journal := map[string]any{
		"schema": "loop-phase-journal/v1", "version": 1, "project": task.Project, "cycle": 7,
		"tasks": []map[string]any{{"task_id": task.ID, "sequence": task.Seq, "generation": task.Generation(), "branch": "codex/timeline", "run_id": runID, "phase": "ci-pending", "updated_at": started.Add(5 * time.Minute)}},
	}
	raw, _ := json.Marshal(journal)
	phasePath := filepath.Join(journalDir, task.Project+"-phases.json")
	if err := os.WriteFile(phasePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	var got deliveryTimelineResponse
	getJSON(t, newHandler(w), "/api/delivery-timeline?task="+task.ID, &got)
	if got.Schema != deliveryTimelineSchema || len(got.Attempts) != 1 {
		t.Fatalf("timeline = %+v", got)
	}
	attempt := got.Attempts[0]
	if attempt.RunID != runID || attempt.Usage.OutputTokens != 25 || attempt.Identity.TaskID != task.ID {
		t.Fatalf("attempt identity/usage = %+v", attempt)
	}
	if currentDeliveryPhase(attempt.Spans) != "ci" {
		t.Fatalf("current phase = %s, spans=%+v", currentDeliveryPhase(attempt.Spans), attempt.Spans)
	}
	for _, span := range attempt.Spans {
		if span.Phase == "verified" && span.Status != "complete" {
			t.Fatalf("successful OK outcome rendered as verification refusal: %+v", span)
		}
	}
	for _, span := range attempt.Spans {
		if span.Phase == "merged" && span.Status != "pending" {
			t.Fatalf("later phase promoted by historical evidence: %+v", span)
		}
		if span.Phase == "ci" && span.DurationMS == nil {
			t.Fatalf("observed current span lacks duration: %+v", span)
		}
	}

	if err := os.WriteFile(phasePath, []byte(`{"schema":"wrong","private":"secret prompt /Users/operator"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	getJSON(t, newHandler(w), "/api/delivery-timeline?task="+task.ID, &got)
	if got.Refusal == "" || !strings.Contains(got.Refusal, "refuses") {
		t.Fatalf("corrupt evidence was not visibly refused: %+v", got)
	}
	body, _ := json.Marshal(got)
	for _, forbidden := range []string{"secret prompt", "/Users/operator"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("private corrupt content leaked into projection: %s", body)
		}
	}
	for _, span := range got.Attempts[0].Spans {
		if phaseRank(span.Phase) > phaseRank("acting") && span.Status != "refused" {
			t.Fatalf("corrupt later phase rendered non-refused: %+v", span)
		}
	}
}

func TestDeliveryTimelineKeepsAttemptsSeparateAndUnknownDurationHonest(t *testing.T) {
	w := dashboardEnv(t)
	task, _ := store.FindTask(w, "002")
	for i, id := range []string{"01TIMELINEOLD0000000000000", "01TIMELINENEW0000000000000"} {
		dir := w.RunDir(id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{RunID: id, Child: "a-worker", Task: task.ID, Role: "implementer", Runtime: "codex", Started: time.Unix(int64(10+i), 0).UTC(), Outcome: "failed"}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := buildDeliveryTimeline(w, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Attempts) != 2 || got.Attempts[0].RunID == got.Attempts[1].RunID {
		t.Fatalf("attempt generations collapsed: %+v", got.Attempts)
	}
	for _, attempt := range got.Attempts {
		if attempt.Generation != 0 || attempt.Identity.TreeSHA != "" {
			t.Fatalf("historical attempt borrowed current-generation identity: %+v", attempt)
		}
		for _, span := range attempt.Spans {
			if span.Phase == "verified" && (span.Status == "complete" || span.Status == "current") {
				t.Fatalf("historical terminal outcome became verified success: %+v", span)
			}
			if span.DurationMS != nil && *span.DurationMS == 0 {
				t.Fatalf("missing timestamp fabricated zero duration: %+v", span)
			}
		}
	}
}

func TestDeliveryTimelineCanonicalEvidenceMatrix(t *testing.T) {
	started := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

	t.Run("correction skipped CI superseded PR and owner handoff fail closed", func(t *testing.T) {
		w, task, runID := timelineEvidenceFixture(t, started, "ci-pending", "OK")
		store.AppendLog(task, "opened https://github.com/mlnomadpy/dacli/pull/41")
		store.AppendLog(task, "superseded by https://github.com/mlnomadpy/dacli/pull/42")
		ev := timelineVerification("commit-current", "tree-current")
		if err := store.AttachExternalVerification(&ev, store.ExternalVerificationEvidence{Provider: "github-check", CheckRunID: "99", HeadSHA: "commit-current", Name: "unit", State: "skipped", SkipReason: "workflow was not triggered"}); err != nil {
			t.Fatal(err)
		}
		if err := store.AppendVerificationEvidence(task, ev); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
		if err := store.WriteReviewTransaction(w, store.ReviewTransaction{Project: task.Project, TaskID: task.ID, Branch: "codex/evidence", State: store.ReviewCorrection, CorrectionTurns: 1, MaxCorrections: 2, CurrentCommit: "commit-current", CurrentTree: "tree-current", UpdatedAt: started.Add(3 * time.Minute)}); err != nil {
			t.Fatal(err)
		}
		handoff := store.RootHandoff{Schema: store.RootHandoffSchema, Version: 1, TaskID: task.ID, RunID: runID, ChildID: "a-worker", Worktree: w.Root, FailedOperation: "publishing /Users/operator/private", FailureClass: "filesystem_sandbox_refusal", NextAction: "owner re-observes it with token=secret-value", CreatedAt: started.Add(4 * time.Minute)}
		raw, _ := json.Marshal(handoff)
		if err := os.WriteFile(store.RootHandoffPathForRun(w, runID), raw, 0o644); err != nil {
			t.Fatal(err)
		}

		got, err := buildDeliveryTimeline(w, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		attempt := got.Attempts[0]
		if len(attempt.PullRequests) != 2 || attempt.PullRequests[0].State != "superseded" || attempt.PullRequests[1].State != "current" {
			t.Fatalf("PR generations = %+v", attempt.PullRequests)
		}
		assertTimelineSpan(t, attempt, "reviewed", "current", "request-changes", 1)
		assertTimelineSpan(t, attempt, "ci", "refused", "check-skipped", 0)
		assertTimelineSpan(t, attempt, "acting", "refused", "filesystem_sandbox_refusal", 0)
		body, _ := json.Marshal(got)
		for _, forbidden := range []string{w.Root, "/Users/operator", "secret-value", "workflow was not triggered"} {
			if strings.Contains(string(body), forbidden) {
				t.Fatalf("private/local evidence leaked %q: %s", forbidden, body)
			}
		}
	})

	t.Run("green exact-head CI and restart recovery remain generation bound", func(t *testing.T) {
		w, task, _ := timelineEvidenceFixture(t, started, "ci-pending", "OK")
		ev := timelineVerification("commit-green", "tree-green")
		if err := store.AttachExternalVerification(&ev, store.ExternalVerificationEvidence{Provider: "github-check", CheckRunID: "100", HeadSHA: "commit-green", Name: "unit", State: "observed", Conclusion: "success", ObservedAt: started.Add(6 * time.Minute)}); err != nil {
			t.Fatal(err)
		}
		if err := store.AppendVerificationEvidence(task, ev); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
		got, err := buildDeliveryTimeline(w, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.Attempts[0].Recovered || got.Attempts[0].Identity.CommitSHA != "commit-green" {
			t.Fatalf("restart identity = %+v", got.Attempts[0])
		}
		assertTimelineSpan(t, got.Attempts[0], "ci", "complete", "checks-green", 0)
	})

	t.Run("failed verification stale review and corrupt chronology never become green", func(t *testing.T) {
		w, task, _ := timelineEvidenceFixture(t, started, "verified", "failed")
		ev := timelineVerification("commit-new", "tree-new")
		if err := store.AppendVerificationEvidence(task, ev); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
		if err := store.WriteReviewTransaction(w, store.ReviewTransaction{Project: task.Project, TaskID: task.ID, Branch: "codex/evidence", State: store.ReviewApproved, MaxCorrections: 2, CurrentCommit: "commit-old", CurrentTree: "tree-old", UpdatedAt: started.Add(2 * time.Minute)}); err != nil {
			t.Fatal(err)
		}
		got, err := buildDeliveryTimeline(w, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertTimelineSpan(t, got.Attempts[0], "verified", "refused", "", 0)
		assertTimelineSpan(t, got.Attempts[0], "reviewed", "refused", "stale-tree", 0)

		writeTimelineJournal(t, w, task, got.Attempts[0].RunID, "verified", started.Add(-time.Minute))
		got, err = buildDeliveryTimeline(w, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertTimelineSpan(t, got.Attempts[0], "verified", "refused", "", 0)
	})

	t.Run("merged but unaccepted names the exact owner action", func(t *testing.T) {
		w, task, _ := timelineEvidenceFixture(t, started, "merged", "OK")
		got, err := buildDeliveryTimeline(w, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertTimelineSpan(t, got.Attempts[0], "merged", "complete", "", 0)
		accepted := spanFor(&got.Attempts[0], "accepted")
		if accepted == nil || accepted.Status != "current" || !strings.Contains(accepted.NextAction, "fresh trunk") {
			t.Fatalf("merged-not-accepted = %+v", accepted)
		}
	})
}

func timelineEvidenceFixture(t *testing.T, started time.Time, phase, outcome string) (*workspace.Workspace, *store.Task, string) {
	t.Helper()
	w := dashboardEnv(t)
	task, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatal(err)
	}
	runID := "01TIMELINEMATRIX00000000000"
	if err := os.MkdirAll(w.RunDir(runID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), procmon.Record{RunID: runID, Child: "a-worker", Task: task.ID, Role: "implementer", Runtime: "codex", Started: started, Outcome: outcome}); err != nil {
		t.Fatal(err)
	}
	writeTimelineJournal(t, w, task, runID, phase, started.Add(5*time.Minute))
	return w, task, runID
}

func writeTimelineJournal(t *testing.T, w *workspace.Workspace, task *store.Task, runID, phase string, at time.Time) {
	t.Helper()
	dir := filepath.Join(w.Root, workspace.Dir, "loop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{
		"schema": "loop-phase-journal/v1", "version": 1, "project": task.Project,
		"tasks": []map[string]any{{"task_id": task.ID, "sequence": task.Seq, "generation": task.Generation(), "run_id": runID, "phase": phase, "updated_at": at}},
	})
	if err := os.WriteFile(filepath.Join(dir, task.Project+"-phases.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func timelineVerification(commit, tree string) store.VerificationEvidence {
	return store.VerificationEvidence{Command: "go test ./...", Argv: []string{"go", "test", "./..."}, ArtifactHash: "sha256:test", Verifier: "a-reviewer", CommitSHA: commit, TreeSHA: tree, ExitCode: 0}
}

func assertTimelineSpan(t *testing.T, attempt deliveryAttemptView, phase, status, verdict string, correction int) {
	t.Helper()
	span := spanFor(&attempt, phase)
	if span == nil || span.Status != status || span.Verdict != verdict || span.Correction != correction {
		t.Fatalf("%s span = %+v, want status=%s verdict=%s correction=%d", phase, span, status, verdict, correction)
	}
}
