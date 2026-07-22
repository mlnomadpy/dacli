package store

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
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

// CalibrationSamples collects every done task with both a three-point
// estimate and a claim→completion span in its Log. Tasks missing either are
// invisible — a hole in the data, not zero effort. Each sample is banded by
// the agent that completed it (role×model×runtime), joined from the run
// records under RunsDir.
func CalibrationSamples(w *workspace.Workspace) []CalibSample {
	tasks, _ := ListTasks(w, "", model.StatusDone)
	bands := runBands(w)
	usage := runUsage(w)
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
			Tokens: usage[t.ID].OutputTokens,
			Band:   bands[t.ID],
		})
	}
	return out
}

// Usage is the token accounting a stream-json run recorded in usage.txt.
type Usage struct {
	OutputTokens int
	InputTokens  int
	NumTurns     int
	CostUSD      float64
}

// runUsage joins each task ID to the token usage of its LAST run that captured
// any (usage.txt written by a usage_format: stream-json runtime). Run dirs are
// walked in ULID (chronological) order, so a later run with usage overwrites an
// earlier one. Tasks whose runs were text runtimes have no usage.txt and are
// simply absent from the map, so calibration falls back to the wall-clock proxy.
func runUsage(w *workspace.Workspace) map[string]Usage {
	out := map[string]Usage{}
	entries, _ := os.ReadDir(w.RunsDir())
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		runDir := filepath.Join(w.RunsDir(), e.Name())
		taskID := runTaskID(runDir)
		if taskID == "" {
			continue
		}
		if u, ok := readUsage(runDir); ok {
			out[taskID] = u
		}
	}
	return out
}

// runTaskID reads the `task:` field from a run's invocation.txt.
func runTaskID(runDir string) string {
	f, err := os.Open(filepath.Join(runDir, "invocation.txt"))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), ":"); ok && strings.TrimSpace(k) == "task" {
			return strings.TrimSpace(v)
		}
	}
	return ""
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

// runBands joins each task ID to its agent band by scanning every run record's
// invocation.txt for `task:`, `role:`, `model:`, and `runtime:`. A task may
// have several runs (supervise turns); os.ReadDir yields run dirs in ULID
// (chronological) order, so overwriting means the LAST — i.e. completing — run
// wins, which is what we want to band the actual by.
func runBands(w *workspace.Workspace) map[string]Band {
	out := map[string]Band{}
	entries, _ := os.ReadDir(w.RunsDir())
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if b, taskID, ok := readRunBand(w, e.Name()); ok {
			out[taskID] = b
		}
	}
	return out
}

// readRunBand parses one run record's invocation.txt into its task ID and band.
// ok is false when the file is missing/unreadable or carries no `task:` line.
func readRunBand(w *workspace.Workspace, runName string) (b Band, taskID string, ok bool) {
	f, err := os.Open(filepath.Join(w.RunsDir(), runName, "invocation.txt"))
	if err != nil {
		return Band{}, "", false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, cut := strings.Cut(sc.Text(), ":")
		if !cut {
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
		}
	}
	return b, taskID, taskID != ""
}

// TaskBand returns the agent band recorded for a task's runs, if any run record
// carries one. It lets the estimate readout pick the empirical distribution
// that actually applies to a specific task.
//
// Rather than building the whole runBands map and discarding all but one key
// (a full RunsDir scan per task — O(tasks×runs) when called in a loop), it
// walks run dirs NEWEST-first (os.ReadDir sorts by ULID name = chronological)
// and stops at the first run naming this task. Newest-first preserves runBands'
// "last completing run wins" semantics while reading only as far as the match.
func TaskBand(w *workspace.Workspace, taskID string) (Band, bool) {
	entries, _ := os.ReadDir(w.RunsDir())
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if !e.IsDir() {
			continue
		}
		b, tID, ok := readRunBand(w, e.Name())
		if !ok || tID != taskID {
			continue
		}
		return b, !b.Empty()
	}
	return Band{}, false
}

// logSpan measures the FINAL claim→completion cycle: the last "completed by"
// stamp and the most recent "claimed by" that preceded it. A task that was
// completed, reopened, re-claimed and re-completed must not report a span that
// stretches from its ORIGINAL claim across the idle gap to the final
// completion — that inflates the wall-clock actual. Log lines are appended
// chronologically, so tracking the running claim and pairing it with each
// completion yields the last real work cycle.
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
