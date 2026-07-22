package store

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Band identifies the AGENT that produced an actual — role × model × runtime.
// D1 calibrates by the band, not by task size: the empirical distribution of a
// band is the honest estimate for the next task that same kind of agent takes
// on. Run records predating the invocation `model:` line have an empty Model,
// so such runs band by role/runtime only.
type Band struct {
	Role, Model, Runtime string
}

// String renders the band as role/model/runtime — the grouping key and the
// display label.
func (b Band) String() string { return b.Role + "/" + b.Model + "/" + b.Runtime }

// Empty reports a band with no attributed run record — the join found nothing.
func (b Band) Empty() bool { return b.Role == "" && b.Model == "" && b.Runtime == "" }

// CalibSample is one completed task's estimate-vs-actual pair — the P2
// capture fields (claim and completion stamps in the Log) finally paying
// rent. Hours is a WALL-CLOCK PROXY. Tokens is the F1 upgrade: when the
// completing run used a usage-reporting runtime (usage_format: stream-json)
// its output-token count is joined here, and tokens — not wall-clock — become
// the real unit. Tokens == 0 means no usage was captured, so consumers fall
// back to the Hours proxy for that sample. Band is the agent
// (role×model×runtime) that did the work, joined from the task's run record;
// it is empty when no run record matches the task.
type CalibSample struct {
	Te     float64
	Hours  float64
	Tokens int
	Band   Band
}

// Ratio is hours-per-point: the empirical wall-clock multiplier (the fallback).
func (s CalibSample) Ratio() float64 { return s.Hours / s.Te }

// HasTokens reports whether this sample carries a real token actual.
func (s CalibSample) HasTokens() bool { return s.Tokens > 0 }

// TokenRatio is output-tokens-per-point: the F1 unit, preferred over Ratio
// whenever HasTokens is true.
func (s CalibSample) TokenRatio() float64 { return float64(s.Tokens) / s.Te }

// MedianTokenRatio is the median output-tokens-per-point (TokenRatio) across the
// samples that fall in band AND carry a real token actual (HasTokens). It is the
// F2 primitive: expected token cost of a spawn = ratio × the task's Te. n is how
// many token-bearing samples the band has — 0 means no token history, so the
// caller falls back to the wall-clock Ratio, and a caller reusing the n≥10
// calibration gate treats a smaller n as provisional. Centralising the filter
// here keeps `spawn --advise` (which DISPLAYS the budget) and the `--max-tokens`
// gate (which ENFORCES it) computing the band's cost the one same way.
func MedianTokenRatio(samples []CalibSample, band Band) (ratio float64, n int) {
	var rs []float64
	for _, s := range samples {
		if s.Band == band && s.HasTokens() {
			rs = append(rs, s.TokenRatio())
		}
	}
	if len(rs) == 0 {
		return 0, 0
	}
	return spm.Median(rs), len(rs)
}

// CalibrationSamples collects every done task with both a three-point
// estimate and a claim→completion span in its Log. Tasks missing either are
// invisible — a hole in the data, not zero effort. Each sample is banded by
// the agent that completed it (role×model×runtime), joined from the run
// records under RunsDir.
func CalibrationSamples(w *workspace.Workspace) []CalibSample {
	return LoadCalibration(w).Samples
}

// Calibration is one walk of RunsDir: the estimate-vs-actual Samples plus a
// per-task agent-band index. A readout that needs both (estimate) builds it
// once instead of re-walking the runs tree — the previous CalibrationSamples
// walked RunsDir twice (bands + usage) and cmdEstimate added a third walk via
// TaskBand.
type Calibration struct {
	Samples []CalibSample
	bands   map[string]Band
}

// TaskBand returns the agent band recorded for a task's runs, if any run
// record carries a non-empty one.
func (c *Calibration) TaskBand(taskID string) (Band, bool) {
	b, ok := c.bands[taskID]
	return b, ok && !b.Empty()
}

// LoadCalibration walks RunsDir ONCE, joining every run's band and token usage
// to its task, then pairs each done task's estimate against its wall-clock (and
// token, when captured) actual.
func LoadCalibration(w *workspace.Workspace) *Calibration {
	recs := runRecords(w)
	bands := make(map[string]Band, len(recs))
	for id, r := range recs {
		bands[id] = r.band
	}
	tasks, _ := ListTasks(w, "", model.StatusDone)
	var out []CalibSample
	for _, t := range tasks {
		tp, ok := t.Estimate()
		if !ok {
			continue
		}
		claimed, done, ok := logSpan(t)
		if !ok {
			continue
		}
		hours := done.Sub(claimed).Hours()
		if hours <= 0 {
			continue
		}
		out = append(out, CalibSample{
			Te:     tp.Expected(),
			Hours:  hours,
			Tokens: recs[t.ID].usage.OutputTokens,
			Band:   recs[t.ID].band,
		})
	}
	return &Calibration{Samples: out, bands: bands}
}

// Usage is the token accounting a stream-json run recorded in usage.txt.
type Usage struct {
	OutputTokens int
	InputTokens  int
	NumTurns     int
	CostUSD      float64
}

// runRecord is one task's joined run data: the agent band and, when a
// usage-reporting runtime captured it, the token usage.
type runRecord struct {
	band  Band
	usage Usage
}

// runRecords walks RunsDir ONCE and joins each task ID to its agent band and
// token usage. Run dirs are read in ULID (chronological) order, so the LAST —
// i.e. completing — run's band wins; usage is carried from the last run that
// captured any (a later text run leaves an earlier stream-json run's tokens
// intact). Merging what were two separate walks (bands + usage), each of which
// re-opened and re-parsed every invocation.txt, halves the I/O per readout.
//
// Two joins are guarded so a run that produced no implementation actual can't
// wipe a real one: (1) an EMPTY band never overwrites a non-empty one (a run
// predating the invocation role/model lines bands empty), and (2) a verify
// panel seat is a CHECK, not an actual — it runs AFTER the completing spawn so
// its ULID is newer, and it must not clobber the implementer's band/usage even
// though it now records a canonical role/model of its own.
func runRecords(w *workspace.Workspace) map[string]runRecord {
	out := map[string]runRecord{}
	entries, _ := os.ReadDir(w.RunsDir())
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		runDir := filepath.Join(w.RunsDir(), e.Name())
		taskID, band, isVerify := readInvocation(runDir)
		if taskID == "" || isVerify {
			continue
		}
		rec := out[taskID]
		if !band.Empty() {
			rec.band = band
		}
		if u, ok := readUsage(runDir); ok {
			rec.usage = u
		}
		out[taskID] = rec
	}
	return out
}

// readInvocation reads a run's invocation.txt in a single pass, returning the
// `task:` id, the agent band (role/model/runtime), and whether the run is a
// verify panel seat (marked by a verify_panel_seat: line) — a check that must
// not be joined as a task's implementation actual.
func readInvocation(runDir string) (taskID string, b Band, isVerify bool) {
	f, err := os.Open(filepath.Join(runDir, "invocation.txt"))
	if err != nil {
		return "", Band{}, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "task":
			taskID = v
		case "role":
			b.Role = v
		case "model":
			b.Model = v
		case "runtime":
			b.Runtime = v
		case "verify_panel_seat":
			isVerify = true
		}
	}
	return taskID, b, isVerify
}

// readUsage parses a run's usage.txt. ok is false when the file is absent (a
// text runtime) or carries no output-token count.
func readUsage(runDir string) (Usage, bool) {
	f, err := os.Open(filepath.Join(runDir, "usage.txt"))
	if err != nil {
		return Usage{}, false
	}
	defer f.Close()
	var u Usage
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "output_tokens":
			u.OutputTokens, _ = strconv.Atoi(v)
		case "input_tokens":
			u.InputTokens, _ = strconv.Atoi(v)
		case "num_turns":
			u.NumTurns, _ = strconv.Atoi(v)
		case "cost_usd":
			u.CostUSD, _ = strconv.ParseFloat(v, 64)
		}
	}
	return u, u.OutputTokens > 0
}

// TaskBand returns the agent band recorded for a task's runs, if any run record
// carries one. It lets the estimate readout pick the empirical distribution
// that actually applies to a specific task. A caller that also needs the
// samples should use LoadCalibration (one walk) rather than pairing this with
// CalibrationSamples (which would walk RunsDir a second time).
func TaskBand(w *workspace.Workspace, taskID string) (Band, bool) {
	b, ok := runRecords(w)[taskID]
	return b.band, ok && !b.band.Empty()
}

// logSpan measures the FINAL claim→completion cycle: the last "completed by"
// stamp and the most recent "claimed by" that preceded it. A task that was
// completed, reopened, re-claimed and re-completed must not report a span that
// stretches from its ORIGINAL claim across the idle gap to the final
// completion — that inflates the wall-clock actual. Log lines are appended
// chronologically, so tracking the running claim and pairing it with each
// completion yields the last real work cycle. (048 correctness fix, preserved
// over 051's LoadCalibration single-walk refactor.)
func logSpan(t *Task) (claimed, done time.Time, ok bool) {
	s, found := t.Doc.Section("Log")
	if !found {
		return
	}
	var pendingClaim time.Time
	for _, line := range strings.Split(s.Content, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ts, err := time.Parse(time.RFC3339, fields[0])
		if err != nil {
			continue
		}
		rest := strings.Join(fields[1:], " ")
		switch {
		case strings.HasPrefix(rest, "claimed by"):
			pendingClaim = ts
		case strings.HasPrefix(rest, "completed by"):
			done = ts
			if !pendingClaim.IsZero() {
				claimed = pendingClaim
			}
		}
	}
	return claimed, done, !claimed.IsZero() && !done.IsZero()
}
