package orchestration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const tokenLedgerSchema = "loop-token-reservations/v1"

// tokenAmount deliberately separates settled spend from reservations. The
// remaining field is nil when provider usage or the runtime limit is advisory:
// printing a number there would turn an estimate into an enforceable promise.
type tokenAmount struct {
	Unit      string `json:"unit"`
	Limit     int64  `json:"limit"`
	Spent     int64  `json:"spent"`
	Reserved  int64  `json:"reserved"`
	Remaining *int64 `json:"remaining"`
}

type runReservation struct {
	Task       string    `json:"task"`
	RunID      string    `json:"run_id,omitempty"`
	Tokens     int64     `json:"tokens"`
	State      string    `json:"state"` // planned, live, settled, or released
	Outcome    string    `json:"outcome,omitempty"`
	Usage      *int64    `json:"usage,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

type tokenBudgetSnapshot struct {
	Schema                 string           `json:"schema"`
	Version                int              `json:"version"`
	Project                string           `json:"project"`
	Cycle                  int              `json:"cycle"`
	Mode                   string           `json:"mode"` // enforceable, advisory, or unknown
	ObservedAt             time.Time        `json:"observed_at"`
	WindowResetAt          *time.Time       `json:"window_reset_at,omitempty"`
	WindowSpentBeforeCycle int64            `json:"window_spent_before_cycle"`
	CycleBudget            tokenAmount      `json:"cycle_budget"`
	RollingBudget          tokenAmount      `json:"rolling_window"`
	Runs                   []runReservation `json:"live_run_reservations"`
	ReviewReservation      int64            `json:"review_reservation"`
	RecoveryReserve        int64            `json:"integration_recovery_reserve"`
	Unallocated            *int64           `json:"unallocated"`
	RequestedWidth         int              `json:"requested_width"`
	AllocatedWidth         int              `json:"allocated_width"`
	UnknownUsageRuns       []string         `json:"unknown_usage_runs,omitempty"`
	AccountingBoundary     string           `json:"accounting_boundary"`
}

func tokenLedgerFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", project+"-tokens.json")
}

func writeTokenBudget(w *workspace.Workspace, snapshot tokenBudgetSnapshot) error {
	snapshot.Schema, snapshot.Version = tokenLedgerSchema, 1
	slices.SortFunc(snapshot.Runs, func(a, b runReservation) int { return strings.Compare(a.Task, b.Task) })
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	path := tokenLedgerFile(w, snapshot.Project)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := writeStateFile(path, string(append(raw, '\n'))); err != nil {
		return err
	}
	got, err := readTokenBudget(w, snapshot.Project)
	if err != nil {
		return fmt.Errorf("validate persisted token reservation ledger: %w", err)
	}
	gotRaw, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		return fmt.Errorf("validate token reservation ledger readback: %w", err)
	}
	if string(gotRaw) != string(raw) {
		return fmt.Errorf("validate persisted token reservation ledger: got %+v, want %+v", got, snapshot)
	}
	return nil
}

func readTokenBudget(w *workspace.Workspace, project string) (tokenBudgetSnapshot, error) {
	raw, err := os.ReadFile(tokenLedgerFile(w, project))
	if err != nil {
		return tokenBudgetSnapshot{}, err
	}
	var snapshot tokenBudgetSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return snapshot, fmt.Errorf("decode token reservation ledger: %w", err)
	}
	if snapshot.Schema != tokenLedgerSchema || snapshot.Version != 1 || snapshot.Project != project {
		return snapshot, fmt.Errorf("invalid token reservation ledger for project %s", project)
	}
	return snapshot, nil
}

func int64ptr(n int64) *int64 { return &n }

func nonnegativeTokens(n int64) int64 {
	if n < 0 {
		return 0
	}
	return n
}

func fewerTokens(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// reservationPlan is the one calculator used by preview and execution. It
// reserves the complete-cycle tail before allocating implementers; shrinking
// therefore never spends the review/integration capacity needed to finish.
func reservationPlan(project string, cycle, requested int, perRun, cycleLimit, windowLimit, windowSpent, existingReserved int64, windowStart time.Time, windowDur time.Duration, enforceable bool, now time.Time) tokenBudgetSnapshot {
	mode := "enforceable"
	if perRun <= 0 {
		mode = "unknown"
	} else if !enforceable {
		mode = "advisory"
	}
	review := perRun
	recovery := perRun / 4
	if recovery > 0 && recovery < 1000 {
		recovery = 1000
	}
	if cycleLimit <= 0 && perRun > 0 {
		cycleLimit = int64(requested)*perRun + review + recovery
	}
	available := cycleLimit - existingReserved
	if windowLimit > 0 {
		rolling := windowLimit - windowSpent - existingReserved
		if rolling < 0 {
			rolling = 0
		}
		if available <= 0 || rolling < available {
			available = rolling
		}
	}
	allocated := requested
	if perRun > 0 && available >= 0 {
		for allocated > 0 && int64(allocated)*perRun+review+recovery > available {
			allocated--
		}
	}
	newReserved := int64(allocated) * perRun
	reserved := newReserved + existingReserved
	unallocated := available - newReserved - review - recovery
	if unallocated < 0 {
		unallocated = 0
	}
	s := tokenBudgetSnapshot{
		Project: project, Cycle: cycle, Mode: mode, ObservedAt: now.UTC(),
		WindowSpentBeforeCycle: windowSpent,
		CycleBudget:            tokenAmount{Unit: "output_tokens", Limit: cycleLimit, Reserved: reserved + review + recovery},
		RollingBudget:          tokenAmount{Unit: "output_tokens", Limit: windowLimit, Spent: windowSpent, Reserved: reserved + review + recovery},
		ReviewReservation:      review, RecoveryReserve: recovery, RequestedWidth: requested, AllocatedWidth: allocated,
		AccountingBoundary: "provider-reported output tokens; reservations are ceilings, not observed billing",
	}
	if mode == "enforceable" {
		if cycleLimit > 0 {
			s.CycleBudget.Remaining = int64ptr(nonnegativeTokens(cycleLimit - s.CycleBudget.Spent - s.CycleBudget.Reserved))
		}
		if windowLimit > 0 {
			s.RollingBudget.Remaining = int64ptr(nonnegativeTokens(windowLimit - windowSpent - s.RollingBudget.Reserved))
		}
		if s.CycleBudget.Remaining != nil || s.RollingBudget.Remaining != nil {
			s.Unallocated = int64ptr(unallocated)
		}
	}
	if windowLimit > 0 && !windowStart.IsZero() {
		reset := windowStart.Add(windowDur).UTC()
		s.WindowResetAt = &reset
	}
	return s
}

func usageForRun(w *workspace.Workspace, runID string) (int64, bool) {
	f, err := os.Open(filepath.Join(w.RunDir(runID), "usage.txt"))
	if err != nil {
		return 0, false
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), ":")
		if strings.TrimSpace(key) != "output_tokens" || !ok {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		return n, err == nil && n >= 0
	}
	return 0, false
}

// reconcileReservations derives terminal transitions from durable proc and
// usage records. Repeating it is idempotent: settled/released entries remain
// terminal, and observed usage replaces (never adds to) a reservation.
func reconcileReservations(w *workspace.Workspace, snapshot tokenBudgetSnapshot, now time.Time) tokenBudgetSnapshot {
	snapshot.UnknownUsageRuns = nil
	var liveReserved int64
	var observedUsage int64
	for i := range snapshot.Runs {
		r := &snapshot.Runs[i]
		if r.State == "released" && r.RunID != "" {
			snapshot.UnknownUsageRuns = append(snapshot.UnknownUsageRuns, r.RunID)
		}
		if r.State == "settled" && r.Usage != nil {
			observedUsage += *r.Usage
		}
		if r.RunID == "" || r.State == "settled" || r.State == "released" {
			if r.State == "live" || r.State == "planned" {
				liveReserved += r.Tokens
			}
			continue
		}
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(r.RunID), "proc.txt"))
		if err != nil || rec.Outcome == "" {
			liveReserved += r.Tokens
			continue
		}
		r.Outcome, r.ObservedAt = rec.Outcome, now.UTC()
		if usage, ok := usageForRun(w, r.RunID); ok {
			r.Usage, r.State = int64ptr(usage), "settled"
			observedUsage += usage
		} else {
			r.State = "released"
			snapshot.UnknownUsageRuns = append(snapshot.UnknownUsageRuns, r.RunID)
		}
	}
	reserved := liveReserved + snapshot.ReviewReservation + snapshot.RecoveryReserve
	if observedUsage > snapshot.CycleBudget.Spent {
		snapshot.CycleBudget.Spent = observedUsage
	}
	if reconstructed := snapshot.WindowSpentBeforeCycle + snapshot.CycleBudget.Spent; reconstructed > snapshot.RollingBudget.Spent {
		snapshot.RollingBudget.Spent = reconstructed
	}
	snapshot.CycleBudget.Reserved, snapshot.RollingBudget.Reserved = reserved, reserved
	if snapshot.Mode == "enforceable" {
		if snapshot.CycleBudget.Limit > 0 {
			snapshot.CycleBudget.Remaining = int64ptr(nonnegativeTokens(snapshot.CycleBudget.Limit - snapshot.CycleBudget.Spent - reserved))
		}
		if snapshot.RollingBudget.Limit > 0 {
			snapshot.RollingBudget.Remaining = int64ptr(nonnegativeTokens(snapshot.RollingBudget.Limit - snapshot.RollingBudget.Spent - reserved))
		}
		switch {
		case snapshot.CycleBudget.Remaining != nil && snapshot.RollingBudget.Remaining != nil:
			snapshot.Unallocated = int64ptr(fewerTokens(*snapshot.CycleBudget.Remaining, *snapshot.RollingBudget.Remaining))
		case snapshot.CycleBudget.Remaining != nil:
			snapshot.Unallocated = int64ptr(*snapshot.CycleBudget.Remaining)
		case snapshot.RollingBudget.Remaining != nil:
			snapshot.Unallocated = int64ptr(*snapshot.RollingBudget.Remaining)
		default:
			snapshot.Unallocated = nil
		}
	} else {
		snapshot.CycleBudget.Remaining, snapshot.RollingBudget.Remaining, snapshot.Unallocated = nil, nil, nil
	}
	snapshot.ObservedAt = now.UTC()
	return snapshot
}
