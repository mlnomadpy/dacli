// Package metrics defines the stable, shared scenario-measurement report.
// Collectors adapt durable workspace records into Run values; every renderer
// consumes Report so missing-data and sample-count semantics cannot drift.
package metrics

import (
	"sort"
	"strings"
	"time"
)

const SchemaVersion = 1

type Window struct {
	Name  string     `json:"name"`
	Since *time.Time `json:"since"`
	Until *time.Time `json:"until"`
}

type Run struct {
	Task              string
	Outcome           string
	Elapsed           time.Duration
	OutputTokens      *int
	TokenBudget       *int
	HumanIntervention *bool
}

type Rate struct {
	Value   *float64 `json:"value"`
	Samples int      `json:"samples"`
}

type Failures struct {
	Classes map[string]int `json:"classes"`
	Samples int            `json:"samples"`
}

type WallTime struct {
	MedianSeconds *float64 `json:"median_seconds"`
	TotalSeconds  *float64 `json:"total_seconds"`
	Samples       int      `json:"samples"`
}

type Tokens struct {
	Output        *int `json:"output"`
	Samples       int  `json:"samples"`
	Budget        *int `json:"budget"`
	BudgetSamples int  `json:"budget_samples"`
}

type Report struct {
	SchemaVersion     int      `json:"schema_version"`
	Window            Window   `json:"window"`
	Runs              int      `json:"runs"`
	TerminalRuns      int      `json:"terminal_runs"`
	Completion        Rate     `json:"completion"`
	Retry             Rate     `json:"retry"`
	Failures          Failures `json:"failures"`
	WallTime          WallTime `json:"wall_time"`
	Tokens            Tokens   `json:"tokens"`
	HumanIntervention Rate     `json:"human_intervention"`
}

func Summarise(window Window, runs []Run) Report {
	r := Report{SchemaVersion: SchemaVersion, Window: window}
	r.Runs = len(runs)
	r.Failures.Classes = map[string]int{}
	perTask := map[string]int{}
	var completed int
	var elapsed []time.Duration
	var outputTokens, tokenBudget, interventions int
	for _, run := range runs {
		if !terminal(run.Outcome) {
			continue
		}
		r.TerminalRuns++
		if successful(run.Outcome) {
			completed++
		} else {
			class := failureClass(run.Outcome)
			r.Failures.Classes[class]++
			r.Failures.Samples++
		}
		if run.Task != "" {
			perTask[run.Task]++
		}
		if run.Elapsed > 0 {
			elapsed = append(elapsed, run.Elapsed)
		}
		if run.OutputTokens != nil {
			outputTokens += *run.OutputTokens
			r.Tokens.Samples++
		}
		if run.TokenBudget != nil {
			tokenBudget += *run.TokenBudget
			r.Tokens.BudgetSamples++
		}
		if run.HumanIntervention != nil {
			if *run.HumanIntervention {
				interventions++
			}
			r.HumanIntervention.Samples++
		}
	}
	if r.TerminalRuns > 0 {
		r.Completion.Value = ratio(completed, r.TerminalRuns)
	}
	if len(perTask) > 0 {
		attempts := 0
		for _, n := range perTask {
			attempts += n
		}
		r.Retry.Value = ratio(attempts-len(perTask), len(perTask))
		r.Retry.Samples = len(perTask)
	}
	if len(elapsed) > 0 {
		sort.Slice(elapsed, func(i, j int) bool { return elapsed[i] < elapsed[j] })
		median := elapsed[len(elapsed)/2].Seconds()
		var total time.Duration
		for _, d := range elapsed {
			total += d
		}
		totalSeconds := total.Seconds()
		r.WallTime = WallTime{MedianSeconds: &median, TotalSeconds: &totalSeconds, Samples: len(elapsed)}
	}
	if r.Tokens.Samples > 0 {
		r.Tokens.Output = &outputTokens
	}
	if r.Tokens.BudgetSamples > 0 {
		r.Tokens.Budget = &tokenBudget
	}
	if r.HumanIntervention.Samples > 0 {
		r.HumanIntervention.Value = ratio(interventions, r.HumanIntervention.Samples)
	}
	r.Completion.Samples = r.TerminalRuns
	return r
}

func ratio(n, d int) *float64 {
	v := float64(n) / float64(d)
	return &v
}

func terminal(outcome string) bool {
	return outcome != "" && outcome != "running" && !strings.HasPrefix(outcome, "running")
}

func successful(outcome string) bool {
	return outcome == "ok" || strings.HasPrefix(outcome, "done")
}

func failureClass(outcome string) string {
	class := strings.TrimSpace(outcome)
	if i := strings.IndexAny(class, ": ("); i >= 0 {
		class = class[:i]
	}
	if class == "" {
		return "unknown"
	}
	return class
}
