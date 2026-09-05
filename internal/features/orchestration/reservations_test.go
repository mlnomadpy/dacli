package orchestration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestReservationPlanFundsCompletionTailBeforeShrinkingWave(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	got := reservationPlan("p", 4, 4, 10_000, 35_000, 100_000, 0, 0, now, 24*time.Hour, true, now)
	if got.AllocatedWidth != 2 {
		t.Fatalf("allocated width = %d, want 2; preview/live must shrink before spawn", got.AllocatedWidth)
	}
	if got.ReviewReservation != 10_000 || got.RecoveryReserve != 2_500 {
		t.Fatalf("completion reserves = review %d recovery %d", got.ReviewReservation, got.RecoveryReserve)
	}
	if got.CycleBudget.Reserved != 32_500 || got.CycleBudget.Remaining == nil || *got.CycleBudget.Remaining != 2_500 || got.Unallocated == nil || *got.Unallocated != 2_500 {
		t.Fatalf("reserved tokens were presented as free: %+v", got.CycleBudget)
	}
	if got.WindowResetAt == nil || !got.WindowResetAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("window reset = %v", got.WindowResetAt)
	}
}

func TestReservationPlanReconstructsConcurrentRunsWithoutDoubleCounting(t *testing.T) {
	now := time.Unix(10_000, 0).UTC()
	first := reservationPlan("p", 2, 3, 10_000, 45_000, 60_000, 5_000, 0, now, time.Hour, true, now)
	if first.AllocatedWidth != 3 {
		t.Fatalf("initial width = %d", first.AllocatedWidth)
	}
	restarted := reservationPlan("p", 2, 3, 10_000, 45_000, 60_000, 5_000, 20_000, now, time.Hour, true, now)
	if restarted.AllocatedWidth != 1 {
		t.Fatalf("restart width = %d, want 1 after two live-run reservations", restarted.AllocatedWidth)
	}
	if restarted.RollingBudget.Reserved != 42_500 {
		t.Fatalf("rolling reserved = %d, want existing 20000 + new 10000 + tail 12500", restarted.RollingBudget.Reserved)
	}
}

func TestPreviewPlanAndLiveWaveUseTheSameAllocatedWidth(t *testing.T) {
	w := loopEnv(t)
	ready := make([]*store.Task, 0, 4)
	refs := map[string]bool{}
	for _, title := range []string{"A", "B", "C", "D"} {
		task, err := store.CreateTask(w, "a-root", "p", title, store.TaskOpts{Accept: []string{"done"}})
		if err != nil {
			t.Fatal(err)
		}
		ready = append(ready, task)
		refs[task.ID] = true
	}
	runner := &fakeRunner{}
	d := newDriver(w, runner, &Governor{WindowTokens: 35_000, WindowDur: time.Hour})
	d.cfg.width, d.cfg.perCycleTok, d.cfg.dryRun = 4, 10_000, true
	if err := d.prepareTokenBudget(ready); err != nil {
		t.Fatal(err)
	}
	if d.tokenBudget.AllocatedWidth != 2 {
		t.Fatalf("preview allocated %d workers, want 2", d.tokenBudget.AllocatedWidth)
	}
	d.runCycle(ready)
	spawned := 0
	for _, call := range runner.calls {
		if len(call) > 0 && call[0] == "spawn" && refs[argAfter(call, "--task")] {
			spawned++
		}
	}
	if spawned != d.tokenBudget.AllocatedWidth {
		t.Fatalf("preview/live accounting diverged: preview=%d live-spawns=%d", d.tokenBudget.AllocatedWidth, spawned)
	}
}

func TestReconcileReservationsSettlesAndReleasesTerminalRunsIdempotently(t *testing.T) {
	w := loopEnv(t)
	now := time.Unix(20_000, 0).UTC()
	writeRun := func(id, outcome string, usage *int64) {
		t.Helper()
		dir := w.RunDir(id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), procmon.Record{RunID: id, Task: "001-a", Outcome: outcome}); err != nil {
			t.Fatal(err)
		}
		if usage != nil {
			if err := os.WriteFile(filepath.Join(dir, "usage.txt"), []byte("output_tokens: "+valueString(usage)+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	// The correction turn exceeded its initial reservation; terminal provider
	// evidence settles the actual instead of clipping it to the estimate.
	used := int64(1_700)
	writeRun("01SETTLED", "completed", &used)
	writeRun("02UNKNOWN", "timed out", nil)
	writeRun("03FAILED", "failed", &used)
	writeRun("04KILLED", "killed", nil)
	s := reservationPlan("p", 1, 0, 1_000, 10_000, 20_000, 0, 0, now, time.Hour, true, now)
	s.Runs = []runReservation{
		{Task: "001-a", RunID: "01SETTLED", Tokens: 1_000, State: "live"},
		{Task: "002-b", RunID: "02UNKNOWN", Tokens: 1_000, State: "live"},
		{Task: "003-c", RunID: "03FAILED", Tokens: 1_000, State: "live"},
		{Task: "004-d", RunID: "04KILLED", Tokens: 1_000, State: "live"},
	}
	once := reconcileReservations(w, s, now)
	twice := reconcileReservations(w, once, now.Add(time.Minute))
	if once.Runs[0].State != "settled" || once.Runs[0].Usage == nil || *once.Runs[0].Usage != 1_700 || once.CycleBudget.Spent != 3_400 {
		t.Fatalf("completed reservation not settled: %+v", once.Runs[0])
	}
	if once.Runs[1].State != "released" || once.Runs[2].State != "settled" || once.Runs[3].State != "released" || len(once.UnknownUsageRuns) != 2 || once.UnknownUsageRuns[0] != "02UNKNOWN" || once.UnknownUsageRuns[1] != "04KILLED" {
		t.Fatalf("unknown terminal usage was not explicit: %+v", once)
	}
	if twice.CycleBudget.Reserved != once.CycleBudget.Reserved || twice.Runs[0].State != "settled" || twice.Runs[1].State != "released" || twice.Runs[2].State != "settled" || twice.Runs[3].State != "released" {
		t.Fatalf("second settlement changed accounting: once=%+v twice=%+v", once, twice)
	}
}

func TestAdvisoryAndUnknownBudgetsNeverRenderEnforceableRemaining(t *testing.T) {
	now := time.Unix(30_000, 0).UTC()
	for _, got := range []tokenBudgetSnapshot{
		reservationPlan("p", 1, 1, 10_000, 30_000, 40_000, 0, 0, now, time.Hour, false, now),
		reservationPlan("p", 1, 1, 0, 0, 0, 0, 0, time.Time{}, time.Hour, false, now),
	} {
		if got.Mode == "enforceable" || got.CycleBudget.Remaining != nil || got.RollingBudget.Remaining != nil || got.Unallocated != nil {
			t.Fatalf("%s accounting rendered enforceable remaining: %+v", got.Mode, got)
		}
	}
	hardCycleOnly := reservationPlan("p", 1, 1, 10_000, 30_000, 0, 0, 0, time.Time{}, time.Hour, true, now)
	if hardCycleOnly.Mode != "enforceable" || hardCycleOnly.CycleBudget.Remaining == nil || hardCycleOnly.RollingBudget.Remaining != nil {
		t.Fatalf("cycle-only hard limit was not represented independently: %+v", hardCycleOnly)
	}
}

func TestCompletedCycleReleasesReviewButRetainsPendingLandingRecovery(t *testing.T) {
	w := loopEnv(t)
	d := newDriver(w, &fakeRunner{}, &Governor{WindowTokens: 50_000, WindowDur: time.Hour})
	d.cfg.perCycleTok = 10_000
	d.tokenBudget = reservationPlan("p", 1, 0, 10_000, 30_000, 50_000, 2_000, 0, d.now(), time.Hour, true, d.now())
	d.pendingLand = []string{"dacli/001-a"}
	d.settleCycleTokenBudget(3_000)
	if d.tokenBudget.ReviewReservation != 0 || d.tokenBudget.RecoveryReserve != 2_500 {
		t.Fatalf("pending landing reserves = review %d recovery %d", d.tokenBudget.ReviewReservation, d.tokenBudget.RecoveryReserve)
	}
	d.pendingLand = nil
	d.settleCycleTokenBudget(3_000)
	if d.tokenBudget.RecoveryReserve != 0 {
		t.Fatalf("completed landing retained %d recovery tokens", d.tokenBudget.RecoveryReserve)
	}
}

func TestWindowRolloverStartsReservationPlanFromFreshSettledSpend(t *testing.T) {
	now := time.Unix(60_000, 0).UTC()
	g := &Governor{WindowTokens: 40_000, WindowDur: time.Hour}
	g.Restore(governorState{WindowStart: now.Add(-2 * time.Hour), WindowSpent: 39_000})
	if decision, _ := g.Before(1, now); decision != Proceed {
		t.Fatalf("rolled window decision = %s", decision)
	}
	got := reservationPlan("p", 2, 1, 10_000, 25_000, g.WindowTokens, g.WindowSpent(), 0, g.WindowStart(), g.windowDur(), true, now)
	if got.RollingBudget.Spent != 0 || got.AllocatedWidth != 1 || got.WindowResetAt == nil || !got.WindowResetAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("fresh-window plan = %+v", got)
	}
}

func TestLoopStatusJSONIncludesCompleteTokenBudget(t *testing.T) {
	w := loopEnv(t)
	now := time.Unix(40_000, 0).UTC()
	if err := writeLoopState(w, loopState{Project: "p", Cycle: 2, WindowTokens: 500, Status: "proceed", UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := writeLoopRecovery(w, loopRecoveryCheckpoint{Project: "p", Cycle: 2, Checkpoint: "cycle-complete", HaltClass: "none", ObservedAt: now}); err != nil {
		t.Fatal(err)
	}
	budget := reservationPlan("p", 3, 2, 1_000, 5_000, 8_000, 500, 0, now, time.Hour, true, now)
	budget.Runs = []runReservation{{Task: "001-a", RunID: "live-run", Tokens: 1_000, State: "live", ObservedAt: now}}
	if err := writeTokenBudget(w, budget); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	if err := cmdLoopStatus(&clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		TokenBudget tokenBudgetSnapshot `json:"token_budget"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("status JSON: %v\n%s", err, out)
	}
	got := payload.TokenBudget
	if got.CycleBudget.Unit != "output_tokens" || got.RollingBudget.Limit != 8_000 || got.WindowResetAt == nil || len(got.Runs) != 1 || got.ReviewReservation == 0 || got.RecoveryReserve == 0 || got.Unallocated == nil {
		t.Fatalf("incomplete token status: %+v", got)
	}
}
