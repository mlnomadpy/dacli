package store

import (
	"bufio"
	"os"
	"path/filepath"
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
// rent. Hours is a WALL-CLOCK PROXY: without runtime usage reporting there
// are no token actuals, and every consumer must label it as the proxy it is.
// Band is the agent (role×model×runtime) that did the work, joined from the
// task's run record; it is empty when no run record matches the task.
type CalibSample struct {
	Te    float64
	Hours float64
	Band  Band
}

// Ratio is hours-per-point: the empirical multiplier.
func (s CalibSample) Ratio() float64 { return s.Hours / s.Te }

// CalibrationSamples collects every done task with both a three-point
// estimate and a claim→completion span in its Log. Tasks missing either are
// invisible — a hole in the data, not zero effort. Each sample is banded by
// the agent that completed it (role×model×runtime), joined from the run
// records under RunsDir.
func CalibrationSamples(w *workspace.Workspace) []CalibSample {
	tasks, _ := ListTasks(w, "", model.StatusDone)
	bands := runBands(w)
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
		out = append(out, CalibSample{Te: tp.Expected(), Hours: hours, Band: bands[t.ID]})
	}
	return out
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
		f, err := os.Open(filepath.Join(w.RunsDir(), e.Name(), "invocation.txt"))
		if err != nil {
			continue
		}
		var taskID string
		var b Band
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
			}
		}
		f.Close()
		if taskID != "" {
			out[taskID] = b
		}
	}
	return out
}

// TaskBand returns the agent band recorded for a task's runs, if any run record
// carries one. It lets the estimate readout pick the empirical distribution
// that actually applies to a specific task.
func TaskBand(w *workspace.Workspace, taskID string) (Band, bool) {
	b, ok := runBands(w)[taskID]
	return b, ok && !b.Empty()
}

// logSpan reads the first "claimed by" and last "completed by" stamps.
func logSpan(t *Task) (claimed, done time.Time, ok bool) {
	s, found := t.Doc.Section("Log")
	if !found {
		return
	}
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
		if strings.HasPrefix(rest, "claimed by") && claimed.IsZero() {
			claimed = ts
		}
		if strings.HasPrefix(rest, "completed by") {
			done = ts
		}
	}
	return claimed, done, !claimed.IsZero() && !done.IsZero()
}
