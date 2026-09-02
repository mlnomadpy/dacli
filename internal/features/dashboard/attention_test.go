package dashboard

import (
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/store"
)

func TestAttentionCollapsePreservesRecurrenceDurationAndDeterministicRank(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	old := now.Add(-3 * time.Hour)
	newer := now.Add(-time.Hour)
	first := attentionCandidate("review_blocked", "high", attentionAffected{Project: "core", Task: "task-1"}, old, false, "review blocked", "repair review", "#/delivery", attentionEvidence{Kind: "review", ID: "one", URL: "#/delivery", ObservedAt: old.Format(time.RFC3339), Confidence: "low"})
	second := attentionCandidate("review_blocked", "high", attentionAffected{Project: "core", Task: "task-1"}, newer, false, "still blocked", "repair review", "#/delivery", attentionEvidence{Kind: "review", ID: "two", URL: "#/delivery", ObservedAt: newer.Format(time.RFC3339), Confidence: "high"})
	critical := attentionCandidate("github_state_unknown", "critical", attentionAffected{Project: "core"}, newer, true, "unknown", "re-observe", "#/agents", attentionEvidence{Kind: "github", ID: "unknown", URL: "#/agents", ObservedAt: newer.Format(time.RFC3339), Confidence: "low"})

	got := collapseAttentionCandidates([]operatorAttentionItem{second, critical, first}, now)
	if len(got) != 2 || got[0].Code != "github_state_unknown" || got[0].Rank != 1 {
		t.Fatalf("deterministic severity order = %+v", got)
	}
	review := got[1]
	if review.Occurrences != 2 || review.FirstObserved != old.Format(time.RFC3339) || review.LastObserved != newer.Format(time.RFC3339) || review.DurationSeconds != int64(2*time.Hour/time.Second) || review.Confidence != "high" {
		t.Fatalf("collapsed recurrence = %+v", review)
	}
	if len(review.Evidence) != 2 || review.RankReason == "" {
		t.Fatalf("evidence/rank explanation = %+v", review)
	}
}

func TestLoopAttentionCoversGovernedPolicyClasses(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	remaining := int64(0)
	operation := loopOperationResponse{
		Project:   "core",
		State:     loopOperationState{Value: "halted-policy", Source: "loop-status", ObservedAt: now.Format(time.RFC3339), HaltClass: "zero-progress", Reason: "second zero progress cycle", NextAction: "inspect blockers"},
		Budget:    loopTokenBudget{Mode: "enforceable", ObservedAt: now.Format(time.RFC3339), Cycle: loopTokenAmount{Remaining: &remaining}},
		Preflight: []loopPreflightPhase{{Phase: "implementation", Task: "task-1", Verdict: "refuse", Evidence: "role capacity exceeded", Remediation: "reduce WIP"}},
		Runs:      []loopOperationRun{{RunID: "run-1", Task: "task-1", State: "handoff-required"}},
	}
	got := loopAttentionCandidates(operation, now)
	want := map[string]bool{"zero_progress": false, "token_reservation_exhausted": false, "wip_capacity_limit": false, "owner_handoff": false}
	for _, alert := range got {
		if _, ok := want[alert.Code]; ok {
			want[alert.Code] = true
		}
	}
	for code, found := range want {
		if !found {
			t.Fatalf("missing governed class %s in %+v", code, got)
		}
	}
}

func TestOperatorAttentionUnknownExternalStateSurvivesRestartAndResolvesCanonically(t *testing.T) {
	w := dashboardEnv(t)
	task, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	evidence := timelineVerification("head-current", "tree-current")
	evidence.Branch = store.TaskBranch(task)
	if err := store.AttachExternalVerification(&evidence, store.ExternalVerificationEvidence{Provider: "github-check", CheckRunID: "99", HeadSHA: "head-current", Name: "required-ci", State: "unobservable", SkipReason: "GitHub API unavailable"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendVerificationEvidence(task, evidence); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}

	first, err := buildOperatorAttention(w, "core", now)
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := buildOperatorAttention(w, "core", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	unknown := findAttention(first.Alerts, "github_state_unknown", task.ID)
	if unknown == nil || unknown.Affected.Task != task.ID || unknown.Retryable != true || unknown.Confidence != "low" || unknown.Evidence[0].ID != "99" {
		t.Fatalf("unknown external alert = %+v", unknown)
	}
	if again := findAttention(restarted.Alerts, "github_state_unknown", task.ID); again == nil || again.ID != unknown.ID {
		t.Fatalf("restart changed or lost alert: first=%+v restart=%+v", unknown, again)
	}

	resolved := timelineVerification("head-current", "tree-current")
	resolved.Branch = store.TaskBranch(task)
	if err := store.AttachExternalVerification(&resolved, store.ExternalVerificationEvidence{Provider: "github-check", CheckRunID: "100", HeadSHA: "head-current", Name: "required-ci", State: "observed", Conclusion: "success", ObservedAt: now.Add(2 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendVerificationEvidence(task, resolved); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	after, err := buildOperatorAttention(w, "core", now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if alert := findAttention(after.Alerts, "github_state_unknown", task.ID); alert != nil {
		t.Fatalf("resolved canonical check remained in queue: %+v", alert)
	}
}

func TestOperatorAttentionAPIIsReadOnlyAndProjectScoped(t *testing.T) {
	w := dashboardEnv(t)
	var response operatorAttentionResponse
	getJSON(t, newHandler(w), "/api/attention?project=core", &response)
	if response.Schema != operatorAttentionSchema || response.Project != "core" || response.Alerts == nil || response.Rule == "" {
		t.Fatalf("attention response = %+v", response)
	}
}

func TestOperatorAttentionKeepsSupersededGenerationHistoricalAndCurrentTreeStale(t *testing.T) {
	w := dashboardEnv(t)
	task, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatal(err)
	}
	oldAt := time.Now().UTC().Add(-time.Hour)
	old := timelineVerification("old-head", "old-tree")
	old.Branch = store.TaskBranch(task)
	if err := store.AttachExternalVerification(&old, store.ExternalVerificationEvidence{Provider: "github-check", CheckRunID: "old-check", HeadSHA: "old-head", Name: "required-ci", State: "unobservable", SkipReason: "old API outage", ObservedAt: oldAt}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendVerificationEvidence(task, old); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := store.CloseTask(w, task, "a-root"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReopenTask(w, task, "a-root", "new delivery generation"); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Add(time.Minute)
	stale, err := buildOperatorAttention(w, "core", now)
	if err != nil {
		t.Fatal(err)
	}
	if findAttention(stale.Alerts, "github_state_unknown", task.ID) != nil {
		t.Fatal("superseded GitHub observation became a current-generation alert")
	}
	if findAttention(stale.Alerts, "verification_stale", task.ID) == nil {
		t.Fatal("old exact-tree verification disappeared instead of remaining explicitly stale")
	}

	current := timelineVerification("new-head", "new-tree")
	current.Branch = store.TaskBranch(task)
	if err := store.AttachExternalVerification(&current, store.ExternalVerificationEvidence{Provider: "github-check", CheckRunID: "new-check", HeadSHA: "new-head", Name: "required-ci", State: "observed", Conclusion: "success", ObservedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendVerificationEvidence(task, current); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteReviewTransaction(w, store.ReviewTransaction{Project: task.Project, TaskID: task.ID, Branch: store.TaskBranch(task), State: store.ReviewApproved, MaxCorrections: 2, CurrentCommit: "new-head", CurrentTree: "new-tree", UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	resolved, err := buildOperatorAttention(w, "core", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if alert := findAttention(resolved.Alerts, "verification_stale", task.ID); alert != nil {
		t.Fatalf("current exact-tree evidence did not resolve stale alert: %+v", alert)
	}
}

func findAttention(alerts []operatorAttentionItem, code, task string) *operatorAttentionItem {
	for i := range alerts {
		if alerts[i].Code == code && alerts[i].Affected.Task == task {
			return &alerts[i]
		}
	}
	return nil
}
