package dashboard

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// writeRunUsage drops a run dir whose ULID name encodes `at` (so the burn series
// buckets it on that day) carrying a usage.txt with `out` output tokens, and —
// when task is non-empty — an invocation.txt joining the run to that task's
// role×model×runtime band (so calibration picks up its tokens).
func writeRunUsage(t *testing.T, w *workspace.Workspace, at time.Time, task string, out int, cost float64) {
	t.Helper()
	dir := w.RunDir(ulid.At(at))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	usage := "output_tokens: " + strconv.Itoa(out) + "\ncost_usd: " + strconv.FormatFloat(cost, 'f', 6, 64) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "usage.txt"), []byte(usage), 0o644); err != nil {
		t.Fatalf("write usage: %v", err)
	}
	if task != "" {
		inv := "task: " + task + "\nrole: builder\nmodel: opus\nruntime: claude\n"
		if err := os.WriteFile(filepath.Join(dir, "invocation.txt"), []byte(inv), 0o644); err != nil {
			t.Fatalf("write invocation: %v", err)
		}
	}
}

// burnEnv builds a workspace with one done, token-bearing calibration sample
// (100 output tokens → the calibrated ceiling) and a much hotter recent run
// (500 tokens today → 5× the ceiling, so the chart must yell).
func burnEnv(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatalf("project: %v", err)
	}
	done, err := store.CreateTask(w, "a-root", "core", "Ship the thing", store.TaskOpts{
		Accept: []string{"it ships"}, Estimate: "1,2,3",
	})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	// Explicit claim→completion span so calibration.logSpan yields a positive
	// wall-clock actual (a same-second span would be dropped).
	done.Doc.SetSection("Log",
		"- 2026-07-20T09:00:00Z claimed by a-root\n- 2026-07-20T11:00:00Z completed by a-root\n")
	if err := store.SaveTask(done); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.MoveTask(w, done, model.StatusDone); err != nil {
		t.Fatalf("move done: %v", err)
	}

	// The completing run: joins the done task to a builder/opus/claude band and
	// reports 100 output tokens — the one calibration sample, so ceiling == 100.
	writeRunUsage(t, w, time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC), done.ID, 100, 0.10)
	// A hot recent run: 500 tokens today, no task join needed (the series counts
	// every usage-bearing run). 500/100 = 5× the ceiling → alert.
	writeRunUsage(t, w, time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC), "", 500, 0.60)

	// A persisted governor snapshot so the window surface has something to read.
	loopDir := filepath.Join(w.Root, workspace.Dir, "loop")
	if err := os.MkdirAll(loopDir, 0o755); err != nil {
		t.Fatalf("loop dir: %v", err)
	}
	gov := "cycle: 3\nwindow_start: 2026-07-24T00:00:00Z\nwindow_spent: 4200\nzero_streak: 0\n"
	if err := os.WriteFile(filepath.Join(loopDir, "core-governor.txt"), []byte(gov), 0o644); err != nil {
		t.Fatalf("governor: %v", err)
	}
	return w
}

func TestAPIBurn(t *testing.T) {
	w := burnEnv(t)
	h := newHandler(w)

	var resp burnResponse
	getJSON(t, h, "/api/burn", &resp)

	if resp.Generated == "" {
		t.Errorf("generated is empty")
	}
	if resp.Unit != "output_tokens" {
		t.Errorf("unit = %q, want output_tokens", resp.Unit)
	}
	if resp.AlertAt != AlertFactor {
		t.Errorf("alert_at = %v, want %v", resp.AlertAt, AlertFactor)
	}

	// Series: two days, chronological, with the newest last.
	if len(resp.Series) != 2 {
		t.Fatalf("series = %d points, want 2\n%+v", len(resp.Series), resp.Series)
	}
	if resp.Series[0].Day != "2026-07-20" || resp.Series[1].Day != "2026-07-24" {
		t.Errorf("series days = %q,%q, want 2026-07-20,2026-07-24", resp.Series[0].Day, resp.Series[1].Day)
	}
	last := resp.Series[1]
	if last.Tokens != 500 || last.Runs != 1 || last.PerRun != 500 {
		t.Errorf("latest point = %+v, want tokens 500 runs 1 per_run 500", last)
	}
	if last.Cost < 0.59 || last.Cost > 0.61 {
		t.Errorf("latest cost_usd = %v, want ~0.60", last.Cost)
	}

	// Ceiling is the calibrated per-run norm (the single 100-token sample).
	if resp.Ceiling != 100 {
		t.Errorf("ceiling = %v, want 100", resp.Ceiling)
	}
	// Rate is the latest day's per-run intensity; ratio and alert follow.
	if resp.Rate != 500 {
		t.Errorf("rate = %v, want 500", resp.Rate)
	}
	if resp.Ratio != 5 {
		t.Errorf("ratio = %v, want 5", resp.Ratio)
	}
	if !resp.Alert {
		t.Errorf("alert = false, want true (rate is 5× the ceiling — the chart must yell)")
	}

	// Bands: the one calibrated agent band, its median tokens, n, not-yet-calibrated.
	if len(resp.Bands) != 1 {
		t.Fatalf("bands = %d, want 1\n%+v", len(resp.Bands), resp.Bands)
	}
	b := resp.Bands[0]
	if b.Band != "builder/opus/claude" || b.Role != "builder" {
		t.Errorf("band identity = %+v", b)
	}
	if b.Expected != 100 || b.N != 1 || b.Calibrated {
		t.Errorf("band = %+v, want expected 100 n 1 calibrated false", b)
	}

	// Windows: the persisted governor snapshot's live spend.
	if len(resp.Windows) != 1 {
		t.Fatalf("windows = %d, want 1\n%+v", len(resp.Windows), resp.Windows)
	}
	win := resp.Windows[0]
	if win.Project != "core" || win.Spent != 4200 {
		t.Errorf("window = %+v, want core spent 4200", win)
	}
	if win.Start != "2026-07-24T00:00:00Z" {
		t.Errorf("window start = %q, want 2026-07-24T00:00:00Z", win.Start)
	}
}

// The same burn payload embedded in /api/state must match the standalone
// /api/burn surface — the two contracts can never drift.
func TestAPIStateEmbedsBurn(t *testing.T) {
	w := burnEnv(t)
	h := newHandler(w)

	var state dashboardState
	getJSON(t, h, "/api/state", &state)
	if len(state.Burn.Series) != 2 {
		t.Errorf("state.burn.series = %d, want 2", len(state.Burn.Series))
	}
	if !state.Burn.Alert || state.Burn.Ceiling != 100 {
		t.Errorf("state.burn = alert %v ceiling %v, want alert true ceiling 100", state.Burn.Alert, state.Burn.Ceiling)
	}
}

// An empty workspace must degrade to a zero-safe burn view: no series, no
// ceiling, no alert — never a divide by zero or a fabricated line.
func TestBurnEmptyWorkspaceIsZeroSafe(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	v, err := buildBurn(w)
	if err != nil {
		t.Fatalf("buildBurn: %v", err)
	}
	if len(v.Series) != 0 || len(v.Bands) != 0 || len(v.Windows) != 0 {
		t.Errorf("empty workspace burn not empty: %+v", v)
	}
	if v.Ceiling != 0 || v.Rate != 0 || v.Ratio != 0 || v.Alert {
		t.Errorf("empty workspace burn not zero: %+v", v)
	}
}

// Below 1.5× the ceiling the chart stays a passive line — no false yell.
func TestBurnNoAlertBelowThreshold(t *testing.T) {
	w := workspaceWithBurn(t, 100, 140) // rate 140 vs ceiling 100 = 1.4× < 1.5×
	v, err := buildBurn(w)
	if err != nil {
		t.Fatalf("buildBurn: %v", err)
	}
	if v.Ceiling != 100 || v.Rate != 140 {
		t.Fatalf("setup wrong: ceiling %v rate %v", v.Ceiling, v.Rate)
	}
	if v.Alert {
		t.Errorf("alert = true at 1.4× the ceiling, want false")
	}
}

// ulidTime is the exact inverse of ulid.At's timestamp half — a round trip
// recovers the millisecond the ULID was minted at.
func TestULIDTimeRoundTrip(t *testing.T) {
	want := time.Date(2026, 7, 24, 15, 4, 5, 123_000_000, time.UTC)
	got, ok := ulidTime(ulid.At(want))
	if !ok {
		t.Fatalf("ulidTime rejected a real ULID")
	}
	if !got.Equal(want) {
		t.Errorf("ulidTime = %v, want %v", got, want)
	}
	if _, ok := ulidTime("not-a-ulid"); ok {
		t.Errorf("ulidTime accepted junk")
	}
}

// workspaceWithBurn builds a workspace whose calibrated ceiling is `ceiling`
// tokens and whose most-recent day burns `rate` tokens in one run.
func workspaceWithBurn(t *testing.T, ceiling, rate int) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatalf("project: %v", err)
	}
	done, err := store.CreateTask(w, "a-root", "core", "Ship it", store.TaskOpts{
		Accept: []string{"x"}, Estimate: "1,2,3",
	})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	done.Doc.SetSection("Log",
		"- 2026-07-20T09:00:00Z claimed by a-root\n- 2026-07-20T11:00:00Z completed by a-root\n")
	if err := store.SaveTask(done); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.MoveTask(w, done, model.StatusDone); err != nil {
		t.Fatalf("move: %v", err)
	}
	writeRunUsage(t, w, time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC), done.ID, ceiling, 0)
	writeRunUsage(t, w, time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC), "", rate, 0)
	return w
}
