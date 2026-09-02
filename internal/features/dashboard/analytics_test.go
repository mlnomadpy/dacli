package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func analyticsEnv(t *testing.T) (*workspace.Workspace, time.Time) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	for i, completed := range []time.Time{now.Add(-48 * time.Hour), now.Add(-12 * time.Hour), now.Add(-35 * 24 * time.Hour)} {
		task, createErr := store.CreateTask(w, "a-root", "core", "Ship measured outcome", store.TaskOpts{Accept: []string{"observable"}, Estimate: "1,2,3"})
		if createErr != nil {
			t.Fatal(createErr)
		}
		task.Doc.SetSection("Log", "- "+completed.Add(-2*time.Hour).Format(time.RFC3339)+" claimed by a-root\n- "+completed.Format(time.RFC3339)+" completed by a-root\n")
		if saveErr := store.SaveTask(task); saveErr != nil {
			t.Fatal(saveErr)
		}
		if moveErr := store.MoveTask(w, task, model.StatusDone); moveErr != nil {
			t.Fatal(moveErr)
		}
		runID := ulid.At(completed.Add(-time.Hour))
		dir := w.RunDir(runID)
		if mkdirErr := os.MkdirAll(dir, 0o755); mkdirErr != nil {
			t.Fatal(mkdirErr)
		}
		if recErr := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{RunID: runID, Task: task.ID, Role: "implementer", Runtime: "codex", Started: completed.Add(-time.Hour), Outcome: "completed"}); recErr != nil {
			t.Fatal(recErr)
		}
		if i != 1 {
			usage := "input_tokens: 10\noutput_tokens: 20\ncost_usd: 0.25\n"
			if writeErr := os.WriteFile(filepath.Join(dir, "usage.txt"), []byte(usage), 0o644); writeErr != nil {
				t.Fatal(writeErr)
			}
		}
		if writeErr := os.WriteFile(filepath.Join(dir, "invocation.txt"), []byte("task: "+task.ID+"\nrole: implementer\nruntime: codex\nmodel: gpt-5.6\n"), 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	return w, now
}

func metricByKey(t *testing.T, got outcomeAnalyticsResponse, key string) analyticsMetric {
	t.Helper()
	for _, metric := range got.Metrics {
		if metric.Key == key {
			return metric
		}
	}
	t.Fatalf("missing metric %s", key)
	return analyticsMetric{}
}

func TestOutcomeAnalyticsKeepsWindowsCostAndHistoricalEvidenceHonest(t *testing.T) {
	w, now := analyticsEnv(t)
	got, err := buildOutcomeAnalytics(w, "core", 30, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != outcomeAnalyticsSchema || got.Performance.TasksScanned != 3 || got.Performance.RunsScanned != 3 {
		t.Fatalf("projection = %+v", got)
	}
	throughput := metricByKey(t, got, "throughput")
	if throughput.Current.Samples != 2 || throughput.Previous.Samples != 1 {
		t.Fatalf("throughput windows = %+v", throughput)
	}
	cost := metricByKey(t, got, "cost")
	if cost.Current.Eligible != 2 || cost.Current.Samples != 1 || cost.Current.State != "advisory" || cost.Current.Coverage != 0.5 {
		t.Fatalf("missing cost became zero/current: %+v", cost.Current)
	}
	if cost.Current.Value == nil || *cost.Current.Value != 0.25 {
		t.Fatalf("cost = %+v", cost.Current.Value)
	}
	tokens := metricByKey(t, got, "tokens")
	if tokens.Current.Samples != 1 || tokens.Current.Eligible != 2 || tokens.Current.State != "partial" {
		t.Fatalf("unknown usage became zero: %+v", tokens.Current)
	}
	ready := metricByKey(t, got, "ready_to_merged")
	if ready.Current.Value != nil || ready.Current.State != "unknown" {
		t.Fatalf("ready proxy fabricated: %+v", ready)
	}
	if len(throughput.Current.Evidence.Tasks) != 2 {
		t.Fatalf("drilldown membership = %+v", throughput.Current.Evidence)
	}
}

func TestOutcomeAnalyticsAPIValidatesRangeAndProject(t *testing.T) {
	w, _ := analyticsEnv(t)
	h := newHandler(w)
	var got outcomeAnalyticsResponse
	getJSON(t, h, "/api/outcomes?project=core&range=7d", &got)
	if got.Current.Days != 7 {
		t.Fatalf("range = %+v", got.Current)
	}
	for _, path := range []string{"/api/outcomes?project=core&range=8d", "/api/outcomes?project=../bad"} {
		req := httptest.NewRequest("GET", path, nil)
		req.Host = "127.0.0.1"
		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, req)
		if rw.Code != 400 {
			t.Fatalf("%s status = %d, want 400", path, rw.Code)
		}
	}
}

func TestOutcomeAnalyticsDoesNotPromotePreReopenVerification(t *testing.T) {
	w, now := analyticsEnv(t)
	tasks, err := store.ListTasks(w, "core", model.StatusDone)
	if err != nil {
		t.Fatal(err)
	}
	task := tasks[0]
	if err := store.AppendVerificationEvidence(task, store.VerificationEvidence{Command: "go test ./...", Argv: []string{"go", "test", "./..."}, WorkingDirectory: w.Root, ExitCode: 0, ArtifactHash: "sha256:old", Verifier: "reviewer", Branch: "old", CommitSHA: "old-commit", TreeSHA: "old-tree", Clean: true, RuntimeVersions: map[string]string{"go": "test"}, ToolVersions: map[string]string{"git": "test"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReopenTask(w, task, "a-root", "regression"); err != nil {
		t.Fatal(err)
	}
	log, _ := task.Doc.Section("Log")
	task.Doc.SetSection("Log", log.Content+"\n- "+now.Add(-2*time.Hour).Format(time.RFC3339)+" PR https://github.com/example/repo/pull/1 superseded\n- "+now.Add(-90*time.Minute).Format(time.RFC3339)+" PR https://github.com/example/repo/pull/2 current\n- "+now.Add(-time.Hour).Format(time.RFC3339)+" completed by a-root\n")
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, task, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	got, err := buildOutcomeAnalytics(w, "core", 30, now)
	if err != nil {
		t.Fatal(err)
	}
	metric := metricByKey(t, got, "current_tree_acceptance")
	if metric.Current.Value == nil || *metric.Current.Value != 0 {
		t.Fatalf("historical verification promoted to current: %+v", metric.Current)
	}
	landing := metricByKey(t, got, "first_pass_landing")
	if landing.Current.Value != nil || landing.Current.State != "unknown" {
		t.Fatalf("incomplete PR history promoted to first-pass success: %+v", landing.Current)
	}
}

func TestAnalyticsWindowsAreHalfOpen(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	if !inWindow(start, start, end) || !inWindow(end.Add(-time.Nanosecond), start, end) || inWindow(end, start, end) {
		t.Fatal("window must include start and exclude end")
	}
}

func TestOutcomeAnalyticsRepresentativeLargeProjectionBudget(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	indexed := make([]analyticsTask, 0, 500)
	runs := make([]analyticsRun, 0, 500)
	for i := 0; i < 500; i++ {
		completed := now.Add(-time.Duration(i%30) * 24 * time.Hour).Add(-time.Minute)
		run := analyticsRun{id: fmt.Sprintf("run-%03d", i), task: fmt.Sprintf("task-%03d", i), role: "implementer", runtime: "codex", model: fmt.Sprintf("model-%d", i%3), outcome: "completed", started: completed.Add(-time.Hour), ended: completed, tokens: 100, usageKnown: true, cost: 0.01, costKnown: true}
		indexed = append(indexed, analyticsTask{task: &store.Task{ID: run.task, Project: "core"}, created: completed.Add(-24 * time.Hour), generation: completed.Add(-24 * time.Hour), completed: completed, runs: []analyticsRun{run}, size: []string{"small", "medium", "large"}[i%3], verificationCurrent: true, verificationContract: "go test ./...", reviewKnown: true})
		runs = append(runs, run)
	}
	started := time.Now()
	response := outcomeAnalyticsResponse{Schema: outcomeAnalyticsSchema, Metrics: buildAnalyticsMetrics(indexed, now.Add(-60*24*time.Hour), now.Add(-30*24*time.Hour), now), Breakdowns: buildAnalyticsBreakdowns(indexed, now.Add(-60*24*time.Hour), now.Add(-30*24*time.Hour), now), Series: buildAnalyticsSeries(indexed, runs, now.Add(-30*24*time.Hour), now)}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("500-task aggregate took %s, budget 250ms", elapsed)
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 256*1024 || len(response.Series) > 31 {
		t.Fatalf("500-task projection bytes=%d series=%d", len(raw), len(response.Series))
	}
}

func TestOutcomeAnalyticsBreakdownMembershipIsExact(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	task := analyticsTask{task: &store.Task{ID: "task-one", Project: "core"}, completed: now.Add(-time.Hour), size: "small", runs: []analyticsRun{{id: "run-a", role: "implementer", runtime: "codex", model: "model-a"}, {id: "run-b", role: "implementer", runtime: "codex", model: "model-b"}}}
	rows := buildAnalyticsBreakdowns([]analyticsTask{task}, now.Add(-60*24*time.Hour), now.Add(-30*24*time.Hour), now)
	for _, row := range rows {
		if row.Dimension == "model" && row.Key == "model-a" {
			if len(row.Evidence.Runs) != 1 || row.Evidence.Runs[0] != "run-a" {
				t.Fatalf("model cohort leaked unrelated run: %+v", row.Evidence)
			}
			return
		}
	}
	t.Fatal("missing model-a cohort")
}

func TestOutcomeAnalyticsCacheAndPayloadAreBounded(t *testing.T) {
	w, now := analyticsEnv(t)
	cache := newOutcomeCache()
	first, err := cache.build(w, "core", 30, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.build(w, "core", 30, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if first.Performance.Cache != "fresh-index" || second.Performance.Cache != "bounded-ttl-hit" {
		t.Fatalf("cache states = %q, %q", first.Performance.Cache, second.Performance.Cache)
	}
	for i := 0; i < analyticsCacheLimit+3; i++ {
		cache.entries[string(rune('a'+i))] = outcomeCacheEntry{created: now.Add(time.Duration(i) * time.Second)}
	}
	if _, err := cache.build(w, "core", 7, now.Add(analyticsCacheTTL+time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(cache.entries) > analyticsCacheLimit {
		t.Fatalf("cache entries = %d, want <= %d", len(cache.entries), analyticsCacheLimit)
	}
	raw, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 256*1024 || len(second.Series) > 90 {
		t.Fatalf("projection too large: bytes=%d series=%d", len(raw), len(second.Series))
	}
	for _, metric := range second.Metrics {
		if len(metric.Current.Evidence.Tasks) > analyticsEvidenceLimit || len(metric.Current.Evidence.Runs) > analyticsEvidenceLimit {
			t.Fatalf("unbounded evidence: %+v", metric.Current.Evidence)
		}
	}
}
