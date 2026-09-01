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
