package insight

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// cmdOverview is the human-first counterpart to `status`: one screen a
// person can read cold on first `cd` into a workspace — who they are, what's
// open across every project, how much is pending, who else is working, and
// a couple of ready-to-run tasks. `status` stays terse, stable, and
// unstyled for agents and scripts; this command is the readable layer on
// top of the same data, colorized when stdout is a real terminal (see
// clikit.Palette) and plain everywhere else — including --json, which is
// refused outright since there is nothing structured to emit here that
// `status`/`agents`/`next` don't already offer machine-readably.
func cmdOverview(ctx *clikit.Ctx, args []string) error {
	if ctx.JSON {
		return clikit.Usagef("overview is a human-readable summary with no --json form — use `status`, `agents`, or `next` for machine output")
	}
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	pal := clikit.NewPalette(ctx)

	fmt.Fprintf(ctx.Stdout, "%s  %s\n", pal.Bold(pal.Cyan("dacli overview")), pal.Dim(fmt.Sprintf("%s (%s)", w.Name, w.ID)))
	if id.Role != "" {
		fmt.Fprintf(ctx.Stdout, "you are %s (grant: %s, role: %s)\n", pal.Bold(id.ID), id.Grant, id.Role)
	} else {
		fmt.Fprintf(ctx.Stdout, "you are %s (grant: %s)\n", pal.Bold(id.ID), id.Grant)
	}

	ps, err := store.ListProjects(w)
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		fmt.Fprintln(ctx.Stdout, "\nno projects yet — `dacli project add \"<title>\"` to create the first one")
		return nil
	}

	fmt.Fprintf(ctx.Stdout, "\n%s\n", pal.Bold("PROJECTS"))
	totalTasks := 0
	for _, p := range ps {
		counts := map[model.Status]int{}
		ts, _ := store.ListTasks(w, p.Slug, "")
		for _, t := range ts {
			counts[t.Status]++
		}
		totalTasks += len(ts)
		slug := pal.Bold(fmt.Sprintf("%-16s", p.Slug)) // pad THEN color: escape codes must never count toward column width
		fmt.Fprintf(ctx.Stdout, "  %s %s  %s  %s  %s  %s\n",
			slug,
			pal.Dim(fmt.Sprintf("○ %d open", counts[model.StatusOpen])),
			pal.Yellow(fmt.Sprintf("▶ %d active", counts[model.StatusActive])),
			pal.Red(fmt.Sprintf("⛔ %d blocked", counts[model.StatusBlocked])),
			pal.Green(fmt.Sprintf("✓ %d done", counts[model.StatusDone])),
			p.Title)
	}
	fmt.Fprintf(ctx.Stdout, "  %s\n", pal.Dim(fmt.Sprintf("%d project(s), %d task(s) total", len(ps), totalTasks)))

	fmt.Fprintf(ctx.Stdout, "\n%s\n", pal.Bold("ACTIVITY"))
	pending, _ := eventlog.List(w, eventlog.Query{Pending: true})
	if len(pending) > 0 {
		fmt.Fprintf(ctx.Stdout, "  pending events   %s\n", pal.Yellow(fmt.Sprintf("%d unsynced — run `dacli sync` as the owner to materialize", len(pending))))
	} else {
		fmt.Fprintf(ctx.Stdout, "  pending events   %s\n", pal.Dim("0"))
	}
	if agents := liveAgentCount(w); agents > 0 {
		fmt.Fprintf(ctx.Stdout, "  live agents      %s\n", pal.Green(fmt.Sprintf("%d running — `dacli agents --tail` to watch", agents)))
	} else {
		fmt.Fprintf(ctx.Stdout, "  live agents      %s\n", pal.Dim("0"))
	}

	if lines := readyNow(w, 3); len(lines) > 0 {
		fmt.Fprintf(ctx.Stdout, "\n%s\n", pal.Bold("READY NOW"))
		for _, l := range lines {
			fmt.Fprintf(ctx.Stdout, "  %s\n", l)
		}
		fmt.Fprintln(ctx.Stdout, pal.Dim("  (`dacli next` for the full ranked list, with critical path)"))
	}

	fmt.Fprintf(ctx.Stdout, "\n%s\n", pal.Bold("USEFUL NEXT"))
	fmt.Fprintln(ctx.Stdout, "  dacli next               what to work on now")
	fmt.Fprintln(ctx.Stdout, "  dacli context <task>     brief an agent on a task")
	fmt.Fprintln(ctx.Stdout, "  dacli status --json      machine-readable snapshot")
	return nil
}

// readyNow returns up to limit short "n. 003-slug  priority" lines for the
// highest-MoSCoW-priority tasks with no unmet non-SS dependency. It mirrors
// cmdNext's readiness rule but deliberately skips CPM/slack — overview wants
// a two-line taste, not the scheduling engine, and keeping this independent
// means overview's rendering can never perturb `next`'s stable, agent-facing
// format.
func readyNow(w *workspace.Workspace, limit int) []string {
	tasks, err := store.ListTasks(w, "", "")
	if err != nil {
		return nil
	}
	done := map[string]bool{}
	byRef := map[string]*store.Task{}
	var open []*store.Task
	for _, t := range tasks {
		for _, ref := range []string{t.ID, strings.TrimPrefix(t.ID, "t-"), t.Slug, fmt.Sprintf("%03d", t.Seq)} {
			byRef[ref] = t
		}
		if t.Status == model.StatusDone {
			done[t.ID] = true
		} else if t.Status != model.StatusBlocked {
			open = append(open, t)
		}
	}
	ready := func(t *store.Task) bool {
		for _, d := range t.Deps() {
			if d.Type == "SS" {
				continue
			}
			if dep, ok := byRef[d.Ref]; ok && !done[dep.ID] {
				return false
			}
		}
		return true
	}
	var cands []*store.Task
	for _, t := range open {
		if ready(t) {
			cands = append(cands, t)
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		pi, pj := model.Priority(cands[i].Priority()).Rank(), model.Priority(cands[j].Priority()).Rank()
		if pi != pj {
			return pi < pj
		}
		return cands[i].Seq < cands[j].Seq
	})
	var out []string
	for i, t := range cands {
		if i >= limit {
			break
		}
		line := fmt.Sprintf("%d. %03d-%s", i+1, t.Seq, t.Slug)
		if p := t.Priority(); p != "" {
			line += "  " + p
		}
		out = append(out, line)
	}
	return out
}

// liveAgentCount mirrors dashboard.liveAgents/execution.liveAgents: read
// every run's proc.txt and keep the ones whose leader process is still
// alive (never trust the file alone — AliveRecord re-probes the PID/start
// pair). Duplicated rather than imported — feature slices never import each
// other (arch_test.go).
func liveAgentCount(w *workspace.Workspace) int {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(e.Name()), "proc.txt"))
		if err != nil {
			continue
		}
		if procmon.AliveRecord(rec) {
			n++
		}
	}
	return n
}
