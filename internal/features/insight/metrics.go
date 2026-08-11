package insight

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// cmdMetrics reports the five figures issue #437 asks for across a repeatable
// window: completion rate, retry rate, wall time, token cost, and
// human-intervention rate.
//
// Every figure is DEFINED in the output. A number whose definition lives in
// someone's head is not a measurement — two readers disagreeing about what
// counts as a retry will disagree about whether the loop is improving, and
// neither will know it.
//
// Derived entirely from what is already on disk (run records, the task store,
// the event log). Nothing new is instrumented, so these numbers describe every
// run this workspace has ever done rather than only the ones after a flag was
// switched on.
func cmdMetrics(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("project", "since"); err != nil {
		return err
	}
	since := time.Time{}
	if s := f.Get("since"); s != "" {
		d, derr := time.ParseDuration(s)
		if derr != nil {
			return clikit.Usagef("--since %q: %v (e.g. 24h, 7d is not valid — use 168h)", s, derr)
		}
		since = time.Now().Add(-d)
	}

	runs := readRuns(w, since)
	if len(runs) == 0 {
		// An empty window is a real answer, and saying "0%" for every rate
		// would be a fabricated one.
		fmt.Fprintln(ctx.Stdout, "no runs in this window — nothing to measure")
		return nil
	}

	m := summarise(w, runs, f.Get("project"))
	m.render(ctx.Stdout)
	return nil
}

// runFacts is one run, read back from its record.
type runFacts struct {
	RunID, Child, Task, Role string
	Outcome                  string
	Elapsed                  time.Duration
	Started                  time.Time
}

// terminal reports whether this run finished. A run still going is excluded
// from every rate: counting it as a non-completion would make a healthy
// in-flight wave look like a failing one.
func (r runFacts) terminal() bool {
	return r.Outcome != "" && r.Outcome != "running" && !strings.HasPrefix(r.Outcome, "running")
}

func (r runFacts) completed() bool {
	return r.Outcome == "ok" || strings.HasPrefix(r.Outcome, "done")
}

func readRuns(w *workspace.Workspace, since time.Time) []runFacts {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return nil
	}
	var out []runFacts
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := w.RunDir(e.Name())
		rec, rerr := procmon.ReadRecord(filepath.Join(dir, "proc.txt"))
		if rerr != nil {
			continue
		}
		if !since.IsZero() && rec.Started.Before(since) {
			continue
		}
		rf := runFacts{RunID: e.Name(), Child: rec.Child, Task: rec.Task, Role: rec.Role, Started: rec.Started}
		raw, oerr := os.ReadFile(filepath.Join(dir, "outcome.md"))
		if oerr == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				k, v, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				switch strings.TrimSpace(k) {
				case "outcome":
					rf.Outcome = strings.TrimSpace(v)
				case "elapsed":
					if d, derr := time.ParseDuration(strings.TrimSpace(v)); derr == nil {
						rf.Elapsed = d
					}
				}
			}
		}
		out = append(out, rf)
	}
	return out
}

type metrics struct {
	Runs, Terminal, Completed int
	TasksAttempted            int
	RunsPerTask               float64
	WallMedian, WallTotal     time.Duration
	Tokens, TokenSamples      int
	Interventions, TasksDone  int
}

func summarise(w *workspace.Workspace, runs []runFacts, project string) metrics {
	var m metrics
	m.Runs = len(runs)
	perTask := map[string]int{}
	var wall []time.Duration
	for _, r := range runs {
		if !r.terminal() {
			continue
		}
		m.Terminal++
		if r.completed() {
			m.Completed++
		}
		if r.Task != "" {
			perTask[r.Task]++
		}
		if r.Elapsed > 0 {
			wall = append(wall, r.Elapsed)
			m.WallTotal += r.Elapsed
		}
	}
	m.TasksAttempted = len(perTask)
	if m.TasksAttempted > 0 {
		total := 0
		for _, n := range perTask {
			total += n
		}
		m.RunsPerTask = float64(total) / float64(m.TasksAttempted)
	}
	if len(wall) > 0 {
		sort.Slice(wall, func(i, j int) bool { return wall[i] < wall[j] })
		m.WallMedian = wall[len(wall)/2]
	}

	// Token cost joins the calibration samples, which already pair a completed
	// task with its run's reported output tokens. Zero means the runtime
	// reported none — said out loud rather than printed as a zero cost.
	for _, s := range store.CalibrationSamples(w) {
		if s.HasTokens() {
			m.Tokens += s.Tokens
			m.TokenSamples++
		}
	}

	// Human intervention: a task the loop could NOT close on its own, so
	// somebody had to adopt it.
	//
	// The signal is "adopted by a-root (owner … orphaned)", not "completed by
	// a-root". Every task closed through `ship` carries the latter, because
	// ship's accept step runs as root by design — counting that would make the
	// rate 100% whenever ship is used, and a figure that cannot vary is not a
	// measurement. Adoption is the narrower, real event: the owning agent
	// finished or died without closing its own task, and a human had to take
	// it over.
	tasks, _ := store.ListTasks(w, project, "done")
	agentTasks := map[string]bool{}
	for _, r := range runs {
		if r.Task != "" && r.Child != "" {
			agentTasks[r.Task] = true
		}
	}
	for _, t := range tasks {
		if !agentTasks[t.ID] {
			continue // never had an agent: a human doing it is not an intervention
		}
		m.TasksDone++
		if store.LogHasStamp(t, "adopted by a-root") {
			m.Interventions++
		}
	}
	return m
}

// render prints each figure WITH its definition. A number whose definition
// lives in someone's head is not a measurement.
func (m metrics) render(w io.Writer) {
	pct := func(n, d int) string {
		if d == 0 {
			return "n/a (nothing terminal in this window)"
		}
		return fmt.Sprintf("%.0f%% (%d of %d)", 100*float64(n)/float64(d), n, d)
	}

	fmt.Fprintf(w, "runs in window:     %d (%d finished, %d still running)\n",
		m.Runs, m.Terminal, m.Runs-m.Terminal)
	fmt.Fprintf(w, "completion rate:    %s\n", pct(m.Completed, m.Terminal))
	fmt.Fprintln(w, "                    a finished run whose outcome is ok/done. A run still")
	fmt.Fprintln(w, "                    going is excluded from every rate below, so a healthy")
	fmt.Fprintln(w, "                    in-flight wave does not read as a failing one.")

	if m.TasksAttempted == 0 {
		fmt.Fprintln(w, "retry rate:         n/a (no run named a task)")
	} else {
		fmt.Fprintf(w, "retry rate:         %.2f runs per task (%d runs over %d task(s))\n",
			m.RunsPerTask, m.Terminal, m.TasksAttempted)
		fmt.Fprintln(w, "                    1.00 means every task was done in one attempt.")
	}

	fmt.Fprintf(w, "wall time:          median %s, total %s\n",
		m.WallMedian.Round(time.Second), m.WallTotal.Round(time.Second))
	fmt.Fprintln(w, "                    per finished run, from spawn to outcome.")

	if m.TokenSamples == 0 {
		fmt.Fprintln(w, "token cost:         NOT CAPTURED — no completed task has a run that")
		fmt.Fprintln(w, "                    reported usage. Add usage_format: stream-json to the")
		fmt.Fprintln(w, "                    runtime; a zero here would be a fabricated number.")
	} else {
		fmt.Fprintf(w, "token cost:         %d output tokens across %d measured task(s)\n", m.Tokens, m.TokenSamples)
		fmt.Fprintln(w, "                    output tokens reported by the runtime, joined to the")
		fmt.Fprintln(w, "                    task its run completed.")
	}

	if m.TasksDone == 0 {
		fmt.Fprintln(w, "intervention rate:  n/a (no agent-worked task closed in this window)")
	} else {
		fmt.Fprintf(w, "intervention rate:  %s\n", pct(m.Interventions, m.TasksDone))
		fmt.Fprintln(w, "                    an agent-worked task the loop could NOT close, so a human")
		fmt.Fprintln(w, "                    adopted it. Counted on \"adopted by\", not \"completed by\":")
		fmt.Fprintln(w, "                    every shipped task carries the latter, since ship accepts")
		fmt.Fprintln(w, "                    as root by design, and a rate stuck at 100% measures")
		fmt.Fprintln(w, "                    nothing. Lower is better.")
	}
}
