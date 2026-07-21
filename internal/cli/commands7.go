// Seventh slice: the supervision loop, the SPM display commands over the
// already-tested engines, threads, and escalation.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
)

// cmdPromptList and cmdPromptShow are the audit surface for the prompt
// registry: every piece of agent-facing prose, listable and inspectable,
// with workspace overrides marked.
func cmdPromptList(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	for _, name := range prompts.Names() {
		_, overridden, _ := prompts.Resolve(w.PromptsDir(), name)
		mark := "embedded"
		if overridden {
			mark = "OVERRIDDEN in .dacli/prompts/"
		}
		fmt.Fprintf(ctx.Stdout, "%-24s %s\n", name, mark)
	}
	return nil
}

func cmdPromptShow(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli prompt show <name>")
	}
	content, overridden, err := prompts.Resolve(w.PromptsDir(), f.pos[0])
	if err != nil {
		return store.ErrNotFound{Ref: "prompt " + f.pos[0]}
	}
	if overridden {
		fmt.Fprintf(ctx.Stderr, "(workspace override)\n")
	}
	fmt.Fprint(ctx.Stdout, content)
	return nil
}

// planned returns an honest stub: what the command is waiting on and where
// the design lives. "not implemented — see DESIGN.md" told nobody anything.
func planned(what, doc string) func(*Ctx, []string) error {
	return func(ctx *Ctx, args []string) error {
		return fmt.Errorf("not built yet: %s. The design is in %s — implementation lands with that subsystem", what, doc)
	}
}

// cmdSupervise runs the RUNTIMES § 7 loop: spawn, evaluate against the
// acceptance criteria written before the work started, correct, repeat. It
// terminates because the criterion is external and turns are capped — the
// exact difference between this and the agent chat the design rejects.
func cmdSupervise(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	taskRef, rtName := f.get("task"), f.get("runtime")
	if taskRef == "" || rtName == "" {
		return usagef("usage: dacli supervise --task <ref> --runtime <name> [--max-turns N] [--grant ro|rw] [--budget N] [--timeout sec] [--cooperative]")
	}
	rt, err := store.LoadRuntime(w, rtName)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath(rt.Binary); err != nil {
		return fmt.Errorf("runtime %s: binary %q not on PATH", rt.Name, rt.Binary)
	}
	t, err := store.FindTask(w, taskRef)
	if err != nil {
		return err
	}

	grant := model.Grant(orDash(f.get("grant"), string(model.GrantRO)))
	sandboxArgs, err := sandboxFor(ctx, rt, grant, f.bool("cooperative"))
	if err != nil {
		return err
	}
	maxTurns := 3
	if n, _ := strconv.Atoi(f.get("max-turns")); n > 0 {
		maxTurns = n
	}
	timeout := 300
	if n, _ := strconv.Atoi(f.get("timeout")); n > 0 {
		timeout = n
	}
	budget, _ := strconv.Atoi(f.get("budget"))

	// ONE child identity across turns: ownership continuity is what lets the
	// child claim on turn 1 and check its own boxes on turn 2.
	childID, token, err := agentid.Spawn(w, id, f.get("role"), grant)
	if err != nil {
		return err
	}

	unmetList := func() []string {
		cur, err := store.FindTask(w, taskRef)
		if err != nil {
			return nil
		}
		var unmet []string
		for _, box := range cur.Acceptance() {
			if !box.Done {
				unmet = append(unmet, box.Text)
			}
		}
		return unmet
	}

	for turn := 1; turn <= maxTurns; turn++ {
		b, err := brief.Assemble(w, taskRef, brief.Options{Budget: budget})
		if err != nil {
			return err
		}
		preamble, perr := protocolPreamble(w, childID, grant, t)
		if perr != nil {
			return perr
		}
		prompt := b.Render() + preamble
		if turn > 1 {
			// No session resume: each turn re-sends the brief plus the
			// correction (templated: prompts/tpl/supervise_correction.md).
			// Announced, per the degradation rule — and turn 3 is a signal
			// the task was mis-sized, not normal operation.
			correction, cerr := prompts.Render(w.PromptsDir(), "supervise_correction", map[string]any{
				"Turn": turn, "MaxTurns": maxTurns, "Unmet": unmetList(),
			})
			if cerr != nil {
				return cerr
			}
			prompt += "\n" + correction
		}
		if turn == 3 {
			fmt.Fprintf(ctx.Stderr, "note: turn 3 — under the small-task doctrine this usually means the task should be decomposed, not retried\n")
		}

		runID := ulid.New()
		runDir := w.RunDir(runID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return err
		}
		_ = os.WriteFile(filepath.Join(runDir, "brief.md"), []byte(prompt), 0o644)
		_ = os.WriteFile(filepath.Join(runDir, "invocation.txt"),
			[]byte(fmt.Sprintf("run: %s\nsupervise_turn: %d/%d\ntask: %s\nchild: %s\nruntime: %s\n", runID, turn, maxTurns, t.ID, childID, rt.Name)), 0o644)

		fmt.Fprintf(ctx.Stderr, "turn %d/%d: %s on %s\n", turn, maxTurns, childID, rt.Name)
		out, elapsed, timedOut, runErr := execRuntime(w.Root, rt, prompt, token, sandboxArgs, timeout)
		_ = os.WriteFile(filepath.Join(runDir, "transcript.log"), out, 0o644)

		// The supervisor owns the objects, so it applies the child's events
		// between turns — claims become ownership, findings become notes.
		if res, err := eventlog.Sync(w, id.ID, id.CanMutate); err == nil && res.Applied > 0 {
			fmt.Fprintf(ctx.Stderr, "  applied %d child event(s)\n", res.Applied)
		}

		unmet := unmetList()
		cur, _ := store.FindTask(w, taskRef)
		outcome := fmt.Sprintf("turn %d: %d unmet, elapsed %s", turn, len(unmet), elapsed)
		_ = os.WriteFile(filepath.Join(runDir, "outcome.md"), []byte(outcome+"\n"), 0o644)

		if len(unmet) == 0 && cur != nil && cur.Status == model.StatusDone {
			fmt.Fprintf(ctx.Stdout, "accepted after %d turn(s): all acceptance criteria met and task done\n", turn)
			return nil
		}
		if timedOut {
			return fmt.Errorf("stalled: turn %d timed out after %ds (run %s)", turn, timeout, runID[:10])
		}
		if runErr != nil {
			fmt.Fprintf(ctx.Stderr, "  turn %d exited non-zero (%v) — child events still count\n", turn, runErr)
		}
	}
	return fmt.Errorf("stalled after %d turns; unmet:\n  - %s\ndecompose the task or fix the criteria — do not simply re-run", maxTurns, strings.Join(unmetList(), "\n  - "))
}

func cmdEstimate(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli estimate <task-ref>")
	}
	t, err := store.FindTask(w, f.pos[0])
	if err != nil {
		return err
	}
	tp, ok := t.Estimate()
	if !ok {
		return fmt.Errorf("%03d-%s has no three-point estimate; add one with --estimate o,m,p — a scalar hides the risk", t.Seq, t.Slug)
	}
	stage := spm.StageElicitation
	if p, err := store.LoadProject(w, t.Project); err == nil && p.Stage != "" {
		stage = spm.Stage(p.Stage)
	}
	e, err := spm.Evaluate(tp, stage)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "%03d-%s\n  three-point: %g / %g / %g\n  Te %.1f · σ %.1f · 1σ range %.1f–%.1f\n  cone (%s): %.1f–%.1f\n",
		t.Seq, t.Slug, tp.Optimistic, tp.Probable, tp.Pessimistic,
		e.Expected, e.Sigma, e.Sigma1Low, e.Sigma1High, stage, e.ConeLow, e.ConeHigh)
	return nil
}

func cmdCriticalPath(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	project := f.get("project")
	if project == "" && len(f.pos) > 0 {
		project = f.pos[0]
	}
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return err
	}
	byRef := map[string]*store.Task{}
	done := map[string]bool{}
	var open []*store.Task
	for _, t := range tasks {
		for _, ref := range []string{t.ID, strings.TrimPrefix(t.ID, "t-"), t.Slug, fmt.Sprintf("%03d", t.Seq)} {
			byRef[ref] = t
		}
		if t.Status == model.StatusDone {
			done[t.ID] = true
		} else {
			open = append(open, t)
		}
	}
	var nodes []spm.Node
	var edges []spm.Edge
	labels := map[string]string{}
	for _, t := range open {
		est, ok := t.Estimate()
		if !ok {
			return fmt.Errorf("%03d-%s has no estimate — CPM needs durations; `dacli next` degrades, this command refuses", t.Seq, t.Slug)
		}
		nodes = append(nodes, spm.Node{ID: t.ID, Duration: est.Expected()})
		labels[t.ID] = fmt.Sprintf("%03d-%s", t.Seq, t.Slug)
		for _, d := range t.Deps() {
			if dep, ok := byRef[d.Ref]; ok && !done[dep.ID] {
				edges = append(edges, spm.Edge{From: dep.ID, To: t.ID, Type: spm.DepType(d.Type)})
			}
		}
	}
	if len(nodes) == 0 {
		fmt.Fprintln(ctx.Stdout, "nothing open")
		return nil
	}
	net, err := spm.ComputeCPM(nodes, edges)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "project duration: %.1f (Te units)\n", net.Duration)
	ordered := make([]spm.Schedule, 0, len(net.Schedules))
	for _, s := range net.Schedules {
		ordered = append(ordered, s)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].EarlyStart < ordered[j].EarlyStart })
	for _, s := range ordered {
		mark := " "
		if s.Critical {
			mark = "★"
		}
		fmt.Fprintf(ctx.Stdout, "%s %-30s Te %5.1f  ES %5.1f  EF %5.1f  slack %5.1f\n",
			mark, labels[s.ID], s.Duration, s.EarlyStart, s.EarlyFinish, s.Slack)
	}
	fmt.Fprintln(ctx.Stdout, "★ = critical path: spawn children here first — slack tasks can wait")
	return nil
}

func cmdWBS(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	project := f.get("project")
	if project == "" && len(f.pos) > 0 {
		project = f.pos[0]
	}
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return err
	}
	children := map[string][]*store.Task{}
	byID := map[string]*store.Task{}
	for _, t := range tasks {
		byID[t.ID] = t
	}
	for _, t := range tasks {
		parent := ""
		if p, ok := t.Doc.Front.Get("parent"); ok {
			parent = strings.TrimSuffix(strings.TrimPrefix(p, "[["), "]]")
		}
		if _, ok := byID[parent]; !ok {
			parent = "" // orphan parents render at root rather than vanishing
		}
		children[parent] = append(children[parent], t)
	}
	var render func(parent string, depth int)
	render = func(parent string, depth int) {
		for _, t := range children[parent] {
			est := ""
			if tp, ok := t.Estimate(); ok {
				est = fmt.Sprintf("  Te %.1f", tp.Expected())
			}
			fmt.Fprintf(ctx.Stdout, "%s%03d-%s [%s]%s\n", strings.Repeat("  ", depth), t.Seq, t.Slug, t.Status, est)
			render(t.ID, depth+1)
		}
	}
	render("", 0)
	return nil
}

// cmdBurndown reports points remaining vs done, with per-day completions
// from the Log stamps. Tokens are unavailable without usage reporting, so
// the time axis is a labeled proxy, not a pretense.
func cmdBurndown(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	tasks, err := store.ListTasks(w, f.get("project"), "")
	if err != nil {
		return err
	}
	var doneP, remP float64
	unestimated := 0
	perDay := map[string]float64{}
	for _, t := range tasks {
		tp, ok := t.Estimate()
		if !ok {
			unestimated++
			continue
		}
		if t.Status == model.StatusDone {
			doneP += tp.Expected()
			if day, ok := completionDay(t); ok {
				perDay[day] += tp.Expected()
			}
		} else {
			remP += tp.Expected()
		}
	}
	fmt.Fprintf(ctx.Stdout, "remaining: %.1f points · done: %.1f points\n", remP, doneP)
	days := make([]string, 0, len(perDay))
	for d := range perDay {
		days = append(days, d)
	}
	sort.Strings(days)
	for _, d := range days {
		fmt.Fprintf(ctx.Stdout, "  %s  %5.1f done\n", d, perDay[d])
	}
	if unestimated > 0 {
		fmt.Fprintf(ctx.Stdout, "(%d task(s) without estimates are invisible here — that is a hole in the chart, not zero work)\n", unestimated)
	}
	return nil
}

func cmdVelocity(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	tasks, err := store.ListTasks(w, "", "")
	if err != nil {
		return err
	}
	perDay := map[string]int{}
	for _, t := range tasks {
		if t.Status != model.StatusDone {
			continue
		}
		if day, ok := completionDay(t); ok {
			perDay[day]++
		}
	}
	if len(perDay) == 0 {
		fmt.Fprintln(ctx.Stdout, "no completions recorded yet")
		return nil
	}
	total := 0
	days := make([]string, 0, len(perDay))
	for d, n := range perDay {
		days = append(days, d)
		total += n
	}
	sort.Strings(days)
	for _, d := range days {
		fmt.Fprintf(ctx.Stdout, "%s  %d task(s)\n", d, perDay[d])
	}
	fmt.Fprintf(ctx.Stdout, "mean %.1f task(s)/active day over %d day(s)\n(time is a proxy — per-token velocity needs runtime usage reporting, which nothing here provides yet)\n",
		float64(total)/float64(len(days)), len(days))
	return nil
}

// completionDay extracts YYYY-MM-DD from the "completed by" Log stamp — the
// capture field paying rent.
func completionDay(t *store.Task) (string, bool) {
	s, ok := t.Doc.Section("Log")
	if !ok {
		return "", false
	}
	for _, line := range strings.Split(s.Content, "\n") {
		if strings.Contains(line, "completed by") {
			fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "- "))
			if len(fields) > 0 && len(fields[0]) >= 10 {
				return fields[0][:10], true
			}
		}
	}
	return "", false
}

func cmdThreads(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	questions, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}})
	if err != nil {
		return err
	}
	answers, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventAnswer}})
	answered := map[string]string{} // about → answering actor (nearest answer after the question)
	for _, a := range answers {
		if _, seen := answered[a.About]; !seen {
			answered[a.About] = a.Actor
		}
	}
	for _, q := range questions {
		status := "OPEN"
		d, _ := os.ReadFile(q.Path)
		if strings.Contains(string(d), "applied: true") {
			status = "answered by " + orDash(answered[q.About])
		}
		firstLine := strings.SplitN(q.Body, "\n", 2)[0]
		fmt.Fprintf(ctx.Stdout, "%s [%s] %s asks about %s: %s\n", q.ID[:10], status, q.Actor, q.About, firstLine)
	}
	if len(questions) == 0 {
		fmt.Fprintln(ctx.Stdout, "no questions asked yet")
	}
	return nil
}

// cmdEscalate is the terminal hop: nothing in the tree owns this, so it
// leaves the tree. --github files an issue via gh — reaching a human where
// they will actually see it, with a notification, outside the session.
func cmdEscalate(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli escalate \"summary\" [--about task] [--github]")
	}
	summary := strings.Join(f.pos, " ")
	about := f.get("about")
	if about != "" {
		if t, err := store.FindTask(w, about); err == nil {
			about = t.ID
		}
	}
	ev, err := eventlog.Append(w, id.ID, model.EventHelp, about, "", "[escalation to human]\n"+summary)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "escalated %s — no role in the tree owns this; a human does now\n", ev.ID[:10])

	if f.bool("github") {
		if _, err := exec.LookPath("gh"); err != nil {
			return fmt.Errorf("--github needs the gh CLI on PATH")
		}
		body := fmt.Sprintf("Escalated from dacli workspace %q by %s.\n\n%s\n\nAnswer with: `dacli answer %s \"...\"`", w.Name, id.ID, summary, ev.ID[:10])
		out, gherr := exec.Command("gh", "issue", "create", "--title", "[dacli] "+summary, "--body", body).Output()
		if gherr != nil {
			return fmt.Errorf("gh issue create failed: %v (the escalation event %s still stands)", gherr, ev.ID[:10])
		}
		fmt.Fprintf(ctx.Stdout, "issue: %s", string(out))
	}
	return nil
}
