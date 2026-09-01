package dashboard

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func writeOperationFixture(t *testing.T, w *workspace.Workspace, name, body string) {
	t.Helper()
	dir := filepath.Join(w.Root, workspace.Dir, "loop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "core"+name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeOperationJSON(t *testing.T, w *workspace.Workspace, name string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeOperationFixture(t, w, name, string(raw))
}

func TestLoopOperationMissingRecordsAreUnknownNotHealthyOrUnlimited(t *testing.T) {
	w := dashboardEnv(t)
	v := buildLoopOperation(w, "core", time.Now())
	if v.Schema != loopOperationSchema || v.State.Value != "not-started" || v.State.Freshness != "missing" {
		t.Fatalf("missing loop state = %+v", v.State)
	}
	if v.Budget.Mode != "unknown" || v.Budget.Cycle.Remaining != nil || v.Budget.Rolling.Remaining != nil || v.Budget.Unallocated != nil {
		t.Fatalf("missing budget rendered as numeric capacity: %+v", v.Budget)
	}
	if !strings.Contains(v.Budget.AccountingBoundary, "unknown, not zero or unlimited") {
		t.Fatalf("missing budget boundary = %q", v.Budget.AccountingBoundary)
	}
}

func TestLoopOperationProjectsPolicyHaltBudgetRoutingAndSafePaths(t *testing.T) {
	w := dashboardEnv(t)
	now := time.Now().UTC().Truncate(time.Second)
	tasks, err := store.ListTasks(w, "core", "open")
	if err != nil || len(tasks) != 1 {
		t.Fatalf("open tasks=%d err=%v", len(tasks), err)
	}
	taskID := tasks[0].ID
	for _, runtime := range []store.Runtime{{Name: "codex-runtime", Binary: "/bin/echo", Harness: "codex"}, {Name: "claude", Binary: "/bin/echo", Harness: "claude"}} {
		if err := store.CreateRuntime(w, "a-root", runtime, "test"); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.CreateRole(w, "a-root", team.Role{Name: "codex-builder", Kind: "implementer", Grant: "rw", Runtime: "codex-runtime", Model: "gpt", WIP: 2, MaxPoints: 5}); err != nil {
		t.Fatal(err)
	}
	profileDir := filepath.Join(w.Root, workspace.Dir, "profiles")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	profile := map[string]any{"version": 1, "project": "core", "routing": map[string]any{"harness_mode": "single", "allowed_harnesses": []string{"codex"}}}
	rawProfile, _ := json.Marshal(profile)
	if err := os.WriteFile(filepath.Join(profileDir, "core.json"), rawProfile, 0o644); err != nil {
		t.Fatal(err)
	}
	writeOperationFixture(t, w, ".txt", "project: core\ncycle: 4\ntrunk_marker: 2\nwindow_tokens: 5000\nbacklog: 1\nstatus: halt\nreason: owner policy refused the wave\nrecovery: \nrollup_landed: 0\nrollup_produced_nothing: 0\nrollup_stalled: 0\nrollup_blocked: 1\nupdated_at: "+now.Format(time.RFC3339)+"\n")
	writeOperationJSON(t, w, "-recovery.json", map[string]any{"schema": "loop-recovery/v1", "version": 1, "project": "core", "cycle": 4, "checkpoint": "complete-cycle-preflight", "halt_class": "policy-refusal", "retryable": false, "next_action": "repair role capacity", "reason": "preflight refused", "observed_at": now})
	writeOperationJSON(t, w, "-phases.json", map[string]any{"schema": "loop-phase-journal/v1", "version": 1, "project": "core", "cycle": 4, "tasks": []map[string]any{{"task_id": taskID, "sequence": 1, "generation": 1, "branch": "task/core-001", "run_id": "01ACTIVE", "phase": "spawned", "updated_at": now}}})
	writeOperationJSON(t, w, "-preflight.json", map[string]any{"schema_version": 2, "project": "core", "cycle": 4, "verdict": "refuse", "classification": "permanent_refusal", "generated_at": now, "phases": []map[string]any{{"phase": "implementation", "task": taskID, "role": "codex-builder", "runtime": "codex-runtime", "model": "gpt", "grant": "rw", "claims": []string{"internal/private.go"}, "capacity": map[string]any{"fits": true, "required_points": 2, "limit_points": 5}, "override": map[string]any{"reason": "owner accepted the bounded delta", "expires_at": now.Add(time.Hour)}, "verdict": "pass", "classification": "pass"}, {"phase": "verification-command", "task": taskID, "working_directory": filepath.Join(w.Root, "internal"), "verdict": "pass", "classification": "pass", "evidence": "go test ./internal/..."}}})
	remaining := int64(999)
	writeOperationJSON(t, w, "-tokens.json", map[string]any{"schema": "loop-token-reservations/v1", "version": 1, "project": "core", "cycle": 4, "mode": "advisory", "observed_at": now, "cycle_budget": map[string]any{"unit": "output_tokens", "limit": 5000, "spent": 1200, "reserved": 2000, "remaining": remaining}, "rolling_window": map[string]any{"unit": "output_tokens", "limit": 10000, "spent": 2200, "reserved": 2000, "remaining": remaining}, "requested_width": 2, "allocated_width": 1, "review_reservation": 500, "integration_recovery_reserve": 250, "unallocated": remaining, "accounting_boundary": "provider-reported output tokens; not billing"})

	v := buildLoopOperation(w, "core", now.Add(10*time.Second))
	if v.State.Value != "halted-policy" || v.State.HaltClass != "permanent_refusal" || v.State.Retryable == nil || *v.State.Retryable {
		t.Fatalf("policy halt = %+v", v.State)
	}
	if v.State.Cycle != 4 || v.State.Generation != 1 {
		t.Fatalf("cycle/generation = %d/%d, want 4/1", v.State.Cycle, v.State.Generation)
	}
	if v.Budget.Mode != "advisory" || v.Budget.Cycle.Remaining != nil || v.Budget.Unallocated != nil {
		t.Fatalf("advisory budget became enforceable: %+v", v.Budget)
	}
	if v.Harness.Mode != "single" || len(v.Harness.Allowed) != 1 || v.Harness.Allowed[0] != "codex" {
		t.Fatalf("harness policy = %+v", v.Harness)
	}
	if len(v.Tasks) != 1 || v.Tasks[0].Role != "codex-builder" || v.Tasks[0].ClaimCount != 1 || v.Tasks[0].VerifyCWD != "internal" || v.Tasks[0].Override == "" || strings.Contains(v.Tasks[0].VerifyCWD, w.Root) {
		t.Fatalf("task operation projection = %+v", v.Tasks)
	}
	if len(v.Routing) != 1 || v.Routing[0].Selected.Role != "codex-builder" {
		t.Fatalf("routing = %+v", v.Routing)
	}
	for _, candidate := range v.Routing[0].Candidates {
		if candidate.Runtime == "claude" {
			t.Fatalf("single-Codex profile advertised Claude fallback: %+v", v.Routing[0])
		}
	}
}

func TestLoopOperationStateVocabulary(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tests := []struct {
		name, status, phase, recovery string
		backlog                       int
		want                          string
	}{
		{name: "active", status: "proceed", backlog: 1, want: "running"},
		{name: "idle", status: "idle", backlog: 1, want: "idle"},
		{name: "sleeping budget", status: "sleep-window", backlog: 1, want: "sleeping-budget"},
		{name: "review", status: "proceed", phase: "review-pending", backlog: 1, want: "waiting-review"},
		{name: "ci", status: "proceed", phase: "ci-pending", backlog: 1, want: "waiting-ci"},
		{name: "recovery unknown", status: "proceed", recovery: "transient-infrastructure-failure", backlog: 1, want: "externally-unknown"},
		{name: "owner handoff", status: "proceed", recovery: "handoff-required", backlog: 1, want: "waiting-owner"},
		{name: "policy recovery", status: "proceed", recovery: "policy-refusal", backlog: 1, want: "halted-policy"},
		{name: "complete", status: "idle", want: "completed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := workspace.Init(t.TempDir(), "a-root")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
				t.Fatal(err)
			}
			writeOperationFixture(t, w, ".txt", fmt.Sprintf("project: core\ncycle: 3\nwindow_tokens: 500\nbacklog: %d\nstatus: %s\nupdated_at: %s\n", tt.backlog, tt.status, now.Format(time.RFC3339)))
			if tt.phase != "" {
				writeOperationJSON(t, w, "-phases.json", map[string]any{"schema": "loop-phase-journal/v1", "version": 1, "project": "core", "cycle": 3, "tasks": []map[string]any{{"task_id": "core/001", "sequence": 1, "generation": 1, "branch": "task/core-001", "phase": tt.phase, "updated_at": now}}})
			}
			if tt.recovery != "" {
				writeOperationJSON(t, w, "-recovery.json", map[string]any{"schema": "loop-recovery/v1", "version": 1, "project": "core", "cycle": 3, "checkpoint": "provider-observation", "halt_class": tt.recovery, "retryable": true, "next_action": "re-observe provider", "reason": "provider state unknown", "observed_at": now})
			}
			got := buildLoopOperation(w, "core", now.Add(10*time.Second))
			if got.State.Value != tt.want {
				t.Fatalf("state = %q, want %q: %+v", got.State.Value, tt.want, got.State)
			}
		})
	}
}

func TestLoopOperationMarksOldStatusStaleWithoutChangingItsMeaning(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	writeOperationFixture(t, w, ".txt", "project: core\ncycle: 3\nwindow_tokens: 500\nbacklog: 1\nstatus: proceed\nupdated_at: "+now.Add(-5*time.Minute).Format(time.RFC3339)+"\n")
	got := buildLoopOperation(w, "core", now)
	if got.State.Value != "running" || got.State.Freshness != "stale" {
		t.Fatalf("stale running state = %+v", got.State)
	}
}

func TestLoopOperationPreservesRestartReservationsAndUnknownUsage(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	writeOperationFixture(t, w, ".txt", "project: core\ncycle: 8\nwindow_tokens: 5000\nbacklog: 1\nstatus: proceed\nupdated_at: "+now.Format(time.RFC3339)+"\n")
	cycleRemaining, rollingRemaining, unallocated := int64(2400), int64(5400), int64(900)
	writeOperationJSON(t, w, "-tokens.json", map[string]any{
		"schema": "loop-token-reservations/v1", "version": 1, "project": "core", "cycle": 8, "mode": "enforceable", "observed_at": now,
		"cycle_budget":          map[string]any{"unit": "output_tokens", "limit": 5000, "spent": 600, "reserved": 2000, "remaining": cycleRemaining},
		"rolling_window":        map[string]any{"unit": "output_tokens", "limit": 10000, "spent": 2600, "reserved": 2000, "remaining": rollingRemaining},
		"live_run_reservations": []map[string]any{{"task": "core/001", "run_id": "01RESTART", "tokens": 1500, "state": "live"}},
		"review_reservation":    300, "integration_recovery_reserve": 200, "unallocated": unallocated, "requested_width": 3, "allocated_width": 1,
		"unknown_usage_runs": []string{"01PREVIOUS"}, "accounting_boundary": "provider-reported output tokens; not billing",
	})
	got := buildLoopOperation(w, "core", now.Add(5*time.Second))
	if got.Wave.Requested != 3 || got.Wave.Allocated != 1 || got.Wave.Live != 1 || got.Budget.ReviewReservation != 300 || got.Budget.RecoveryReserve != 200 {
		t.Fatalf("restart reservations = wave %+v budget %+v", got.Wave, got.Budget)
	}
	if len(got.Budget.UnknownUsageRuns) != 1 || got.Budget.UnknownUsageRuns[0] != "01PREVIOUS" || got.Budget.Cycle.Remaining == nil || *got.Budget.Cycle.Remaining != cycleRemaining {
		t.Fatalf("unknown usage/accounting = %+v", got.Budget)
	}
}

func TestLoopOperationPartialEvidenceIsNotFreshOrCorrupt(t *testing.T) {
	v := loopOperationResponse{State: loopOperationState{Value: "running", Freshness: "fresh"}}
	markLoopPartial(&v)
	if v.State.Value != "running" || v.State.Freshness != "partial" {
		t.Fatalf("partial evidence = %+v", v.State)
	}
	markLoopCorrupt(&v, "phase journal", errors.New("bad bytes"))
	markLoopPartial(&v)
	if v.State.Value != "corrupt" || v.State.Freshness != "corrupt" {
		t.Fatalf("partial overwrote corruption = %+v", v.State)
	}
}

func TestLoopOperationCorruptBudgetFailsClosed(t *testing.T) {
	w := dashboardEnv(t)
	now := time.Now().UTC().Truncate(time.Second)
	writeOperationFixture(t, w, ".txt", "project: core\ncycle: 1\nwindow_tokens: 100\nbacklog: 1\nstatus: proceed\nupdated_at: "+now.Format(time.RFC3339)+"\n")
	writeOperationFixture(t, w, "-tokens.json", `{"schema":"wrong","project":"core","mode":"enforceable"}`)
	v := buildLoopOperation(w, "core", now)
	if v.State.Value != "corrupt" || v.State.Freshness != "corrupt" || v.State.Retryable == nil || *v.State.Retryable {
		t.Fatalf("corrupt ledger presented as healthy: %+v", v.State)
	}
	if len(v.Warnings) == 0 || !strings.Contains(v.Warnings[0], "token reservation ledger") {
		t.Fatalf("corrupt warning = %v", v.Warnings)
	}
}

func TestLoopOperationImpossibleBudgetValuesFailClosed(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	writeOperationFixture(t, w, ".txt", "project: core\ncycle: 1\nwindow_tokens: 100\nbacklog: 1\nstatus: proceed\nupdated_at: "+now.Format(time.RFC3339)+"\n")
	writeOperationJSON(t, w, "-tokens.json", map[string]any{
		"schema": "loop-token-reservations/v1", "version": 1, "project": "core", "cycle": 1, "mode": "enforceable", "observed_at": now,
		"cycle_budget":    map[string]any{"unit": "output_tokens", "limit": 100, "spent": 10, "reserved": -1},
		"rolling_window":  map[string]any{"unit": "output_tokens", "limit": 100, "spent": 10, "reserved": 0},
		"requested_width": 1, "allocated_width": 2,
	})
	got := buildLoopOperation(w, "core", now)
	if got.State.Value != "corrupt" || got.State.Retryable == nil || *got.State.Retryable {
		t.Fatalf("impossible budget values = %+v", got.State)
	}
}

func TestAPILoopOperationRequiresSelectedProject(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)
	if code := do(t, h, http.MethodGet, "/api/loop-operation", "localhost"); code != http.StatusBadRequest {
		t.Fatalf("unscoped loop operation = %d, want 400", code)
	}
	var response loopOperationResponse
	getJSON(t, h, "/api/loop-operation?project=core", &response)
	if response.Project != "core" || response.Schema != loopOperationSchema {
		t.Fatalf("response = %+v", response)
	}
}
