package store

import (
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// CalibSample is one completed task's estimate-vs-actual pair — the P2
// capture fields (claim and completion stamps in the Log) finally paying
// rent. Hours is a WALL-CLOCK PROXY: without runtime usage reporting there
// are no token actuals, and every consumer must label it as the proxy it is.
type CalibSample struct {
	Te    float64
	Hours float64
}

// Ratio is hours-per-point: the empirical multiplier.
func (s CalibSample) Ratio() float64 { return s.Hours / s.Te }

// CalibrationSamples collects every done task with both a three-point
// estimate and a claim→completion span in its Log. Tasks missing either are
// invisible — a hole in the data, not zero effort.
func CalibrationSamples(w *workspace.Workspace) []CalibSample {
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
		out = append(out, CalibSample{Te: tp.Expected(), Hours: hours})
	}
	return out
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
