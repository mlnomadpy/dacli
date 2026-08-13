package insight

import (
	"fmt"
	"io"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	metricspkg "github.com/mlnomadpy/dacli/internal/metrics"
)

func cmdMetrics(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("project", "since", "name"); err != nil {
		return err
	}
	window := metricspkg.Window{Name: f.Get("name")}
	if window.Name == "" {
		window.Name = "current"
	}
	if value := f.Get("since"); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return clikit.Usagef("--since %q: %v (e.g. 24h, 7d is not valid — use 168h)", value, err)
		}
		since := time.Now().Add(-duration)
		until := time.Now()
		window.Since, window.Until = &since, &until
	}
	report := metricspkg.Collect(w, window, f.Get("project"))
	if ctx.JSON {
		return clikit.EmitJSON(ctx, report)
	}
	renderMetrics(ctx.Stdout, report)
	return nil
}

func renderMetrics(w io.Writer, report metricspkg.Report) {
	pct := func(metric metricspkg.Rate) string {
		if metric.Value == nil {
			return "n/a (no measured samples)"
		}
		return fmt.Sprintf("%.0f%% (%d sample(s))", 100**metric.Value, metric.Samples)
	}
	fmt.Fprintf(w, "scenario window:    %s\n", report.Window.Name)
	fmt.Fprintf(w, "runs in window:     %d (%d finished, %d still running)\n", report.Runs, report.TerminalRuns, report.Runs-report.TerminalRuns)
	fmt.Fprintf(w, "completion rate:    %s\n", pct(report.Completion))
	fmt.Fprintln(w, "                    a finished run whose outcome is ok/done; in-flight runs are excluded.")
	if report.Retry.Value == nil {
		fmt.Fprintln(w, "retry rate:         n/a (no finished run named a task)")
	} else {
		fmt.Fprintf(w, "retry rate:         %.2f retries per task (%d task sample(s))\n", *report.Retry.Value, report.Retry.Samples)
		fmt.Fprintln(w, "                    0.00 means every task was done in one attempt.")
	}
	if report.WallTime.MedianSeconds == nil {
		fmt.Fprintln(w, "wall time:          NOT CAPTURED (0 samples)")
	} else {
		fmt.Fprintf(w, "wall time:          median %s, total %s (%d samples)\n", time.Duration(*report.WallTime.MedianSeconds*float64(time.Second)).Round(time.Second), time.Duration(*report.WallTime.TotalSeconds*float64(time.Second)).Round(time.Second), report.WallTime.Samples)
	}
	fmt.Fprintln(w, "                    per finished run, from spawn to outcome.")
	if report.Tokens.Output == nil {
		fmt.Fprintln(w, "token budget:       NOT CAPTURED (0 samples) — a zero would be fabricated.")
	} else {
		fmt.Fprintf(w, "token usage:        %d output tokens (%d measured run(s))\n", *report.Tokens.Output, report.Tokens.Samples)
	}
	fmt.Fprintln(w, "                    output-token usage reported by the runtime.")
	if report.Tokens.Budget == nil {
		fmt.Fprintln(w, "token budget:       NOT CAPTURED (0 samples)")
	} else {
		fmt.Fprintf(w, "token budget:       %d tokens across %d measured run(s)\n", *report.Tokens.Budget, report.Tokens.BudgetSamples)
	}
	if report.Failures.Samples == 0 {
		fmt.Fprintln(w, "failure classes:    none (0 failed run samples)")
	} else {
		fmt.Fprintf(w, "failure classes:    %v (%d failed run samples)\n", report.Failures.Classes, report.Failures.Samples)
	}
	fmt.Fprintln(w, "                    terminal non-ok/done outcomes grouped by recorded class.")
	fmt.Fprintf(w, "intervention rate:  %s\n", pct(report.HumanIntervention))
	fmt.Fprintln(w, "                    an agent-worked task adopted by a-root before completion.")
}
