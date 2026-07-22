// Package insight is the read-only-views slice: status, scheduling views
// over the SPM engines, quality checks, anti-pattern detection, and the
// standup roll-up. Nothing here mutates the workspace.
package insight

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

var Commands = []clikit.Command{
	{Path: "status", Brief: "Tree-wide project state in one screen", Run: cmdStatus},
	{Path: "lint", Brief: "Format, INVEST, requirements-quality, and ambiguity checks", Run: cmdLint},
	{Path: "next", Brief: "What to work on now: MoSCoW, then critical path (--parallel N)", Run: cmdNext},
	{Path: "estimate", Brief: "PERT three-point estimate widened by the Cone of Uncertainty", Run: cmdEstimate},
	{Path: "critical-path", Brief: "CPM: full schedule with slack; star marks the critical path", Run: cmdCriticalPath},
	{Path: "wbs", Brief: "Work breakdown tree (task add --parent builds it)", Run: cmdWBS},
	{Path: "burndown", Brief: "Points remaining vs done, per-day completions", Run: cmdBurndown},
	{Path: "velocity", Brief: "Completions per active day (time proxy until usage reporting)", Run: cmdVelocity},
	{Path: "calibrate", Brief: "Te vs actuals: the empirical multiplier by size band (P2)", Run: cmdCalibrate},
	{Path: "taint", Brief: "Blast radius of a suspect source over event/note origins (P4)", Run: cmdTaint},
	{Path: "doctor", Brief: "Detect management anti-patterns in tasks, risks, and the log", Run: cmdDoctor},
	{Path: "standup", Brief: "Per-agent roll-up: done, doing, impediments — derived, never filed", Run: cmdStandup},
}

func cmdStatus(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	ps, err := store.ListProjects(w)
	if err != nil {
		return err
	}
	for _, p := range ps {
		counts := map[model.Status]int{}
		ts, _ := store.ListTasks(w, p.Slug, "")
		for _, t := range ts {
			counts[t.Status]++
		}
		fmt.Fprintf(ctx.Stdout, "%-16s open:%d active:%d blocked:%d done:%d  %s\n",
			p.Slug, counts[model.StatusOpen], counts[model.StatusActive],
			counts[model.StatusBlocked], counts[model.StatusDone], p.Title)
	}
	pending, _ := eventlog.List(w, eventlog.Query{Pending: true})
	if len(pending) > 0 {
		fmt.Fprintf(ctx.Stdout, "pending events: %d (run `dacli sync` as the owner to materialize)\n", len(pending))
	}
	return nil
}

// cmdLint applies the asymmetric scope policy from SPM.md: titles and
// acceptance at moderate-and-above, bodies at major only.
func cmdLint(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	var tasks []*store.Task
	if len(f.Pos) > 0 {
		t, err := store.FindTask(w, f.Pos[0])
		if err != nil {
			return err
		}
		tasks = []*store.Task{t}
	} else {
		tasks, err = store.ListTasks(w, f.Get("project"), "")
		if err != nil {
			return err
		}
	}

	total := 0
	for _, t := range tasks {
		report := func(where string, finds []spm.Finding) {
			for _, fd := range finds {
				total++
				fmt.Fprintf(ctx.Stdout, "%03d-%s %s: %s\n", t.Seq, t.Slug, where, fd)
			}
		}
		report("title", spm.Scan(t.Title, spm.Options{}))
		for _, box := range t.Acceptance() {
			report("acceptance", spm.Scan(box.Text, spm.Options{}))
		}
		for _, s := range t.Doc.Sections {
			if s.Level > 1 && !strings.EqualFold(s.Title, "Acceptance") && !strings.EqualFold(s.Title, "Log") {
				report("body", spm.Scan(s.Content, spm.Options{MinSeverity: spm.SevMajor}))
			}
		}
		if t.Status != model.StatusDone && len(t.Acceptance()) == 0 {
			total++
			fmt.Fprintf(ctx.Stdout, "%03d-%s INVEST: no acceptance criteria — the agent cannot know when to stop\n", t.Seq, t.Slug)
		}
	}
	if total == 0 {
		fmt.Fprintln(ctx.Stdout, "clean")
	}
	return nil
}

// cmdNext: MoSCoW first, then the critical path — which tasks to spawn
// children on FIRST; fanning out onto slack tasks while the critical path
// idles is the default agent failure.
func cmdNext(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	limit := 3
	if n := f.Get("parallel"); n != "" {
		fmt.Sscanf(n, "%d", &limit)
	}

	tasks, err := store.ListTasks(w, f.Get("project"), "")
	if err != nil {
		return err
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
	if len(open) == 0 {
		fmt.Fprintln(ctx.Stdout, "nothing open")
		return nil
	}

	// ready: every non-SS dependency is done. SS permits overlap.
	ready := func(t *store.Task) bool {
		for _, d := range t.Deps() {
			if d.Type == "SS" {
				continue
			}
			dep, ok := byRef[d.Ref]
			if ok && !done[dep.ID] {
				return false
			}
		}
		return true
	}

	// CPM needs durations; degrade to MoSCoW-then-sequence when estimates
	// are missing, and SAY SO.
	slack := map[string]float64{}
	haveCPM := true
	var nodes []spm.Node
	var edges []spm.Edge
	for _, t := range open {
		est, ok := t.Estimate()
		if !ok {
			haveCPM = false
			break
		}
		nodes = append(nodes, spm.Node{ID: t.ID, Duration: est.Expected()})
		for _, d := range t.Deps() {
			if dep, ok := byRef[d.Ref]; ok && !done[dep.ID] {
				edges = append(edges, spm.Edge{From: dep.ID, To: t.ID, Type: spm.DepType(d.Type)})
			}
		}
	}
	if haveCPM {
		net, err := spm.ComputeCPM(nodes, edges)
		if err != nil {
			return fmt.Errorf("dependency graph: %w", err)
		}
		for id, s := range net.Schedules {
			slack[id] = s.Slack
		}
	} else {
		fmt.Fprintln(ctx.Stderr, "note: estimates missing — falling back to MoSCoW-then-sequence order, no critical path")
	}

	var cands []*store.Task
	for _, t := range open {
		if ready(t) {
			cands = append(cands, t)
		}
	}
	if len(cands) == 0 {
		fmt.Fprintln(ctx.Stdout, "no task is ready: everything open is waiting on a dependency")
		return nil
	}
	sort.SliceStable(cands, func(i, j int) bool {
		pi, pj := model.Priority(cands[i].Priority()).Rank(), model.Priority(cands[j].Priority()).Rank()
		if pi != pj {
			return pi < pj
		}
		if haveCPM && slack[cands[i].ID] != slack[cands[j].ID] {
			return slack[cands[i].ID] < slack[cands[j].ID]
		}
		return cands[i].Seq < cands[j].Seq
	})

	// Never recommend a could while a must is ready.
	top := model.Priority(cands[0].Priority()).Rank()
	n := 0
	for _, t := range cands {
		if model.Priority(t.Priority()).Rank() != top || n >= limit {
			break
		}
		line := fmt.Sprintf("%d. %03d-%s", n+1, t.Seq, t.Slug)
		if p := t.Priority(); p != "" {
			line += "  " + p
		}
		if haveCPM {
			if slack[t.ID] == 0 {
				line += "  · critical path"
			} else {
				line += fmt.Sprintf("  · slack %.1f", slack[t.ID])
			}
		}
		fmt.Fprintln(ctx.Stdout, line)
		n++
	}

	// Scope-matched lessons (D2): a cross-project lesson whose topic overlaps a
	// task we just suggested, and which points at a role, is a HINT about who to
	// spawn on it — surfaced so the operator does not have to re-derive from the
	// log what a prior lesson already learned. It never assigns (axiom 3: the
	// model still chooses); it only annotates the tasks shown above.
	if n > 0 {
		if lessons := store.WorkspaceLessons(w, ""); len(lessons) > 0 {
			roles, _ := store.LoadRoles(w)
			for _, t := range cands[:n] {
				for _, l := range lessons {
					if !lessonMatchesTask(l, t) {
						continue
					}
					if role := roleForLesson(roles, l); role != "" {
						fmt.Fprintf(ctx.Stdout, "   ↳ lesson %q (%s) applies to %03d-%s — consider role %s\n",
							l.Title, l.Project, t.Seq, t.Slug, role)
					} else {
						fmt.Fprintf(ctx.Stdout, "   ↳ lesson %q (%s) applies to %03d-%s\n",
							l.Title, l.Project, t.Seq, t.Slug)
					}
				}
			}
		}
	}

	for _, t := range open {
		if !ready(t) && model.Priority(t.Priority()).Rank() < top {
			fmt.Fprintf(ctx.Stderr, "note: %03d-%s (%s) outranks these but is waiting on a dependency\n", t.Seq, t.Slug, t.Priority())
		}
	}
	return nil
}

// lessonMatchesTask reports topical overlap between a cross-project lesson and
// a task: a shared significant word between the lesson's title/body and the
// task's title/slug. Deliberately crude, like the lessons channel it reads from
// (store.WorkspaceLessons) — a spurious hint costs one ignorable line, a missed
// one costs a re-derivation.
func lessonMatchesTask(l store.Lesson, t *store.Task) bool {
	hay := strings.ToLower(l.Title + " " + l.Body)
	for w := range significantWords(t.Title + " " + strings.ReplaceAll(t.Slug, "-", " ")) {
		if strings.Contains(hay, w) {
			return true
		}
	}
	return false
}

// roleForLesson maps a lesson to the role it points at: first a SCOPE match —
// a path the lesson cites that falls inside a role's declared boundary — then a
// name match if the lesson names a role outright. Empty when the lesson maps to
// no role, in which case the hint still fires without a suggestion.
func roleForLesson(roles []team.Role, l store.Lesson) string {
	for _, tok := range pathTokens(l.Body + " " + l.Title) {
		for _, r := range roles {
			if len(r.Scope) > 0 && r.InScope(tok) {
				return r.Name
			}
		}
	}
	text := strings.ToLower(l.Title + " " + l.Body)
	for _, r := range roles {
		if r.Name != "" && strings.Contains(text, strings.ToLower(r.Name)) {
			return r.Name
		}
	}
	return ""
}

// pathTokens pulls path-like tokens (a slash, or a .go suffix) out of free
// text, stripping the file: prefix and :line suffix that findings use, so they
// can be tested against a role's scope globs.
func pathTokens(s string) []string {
	var out []string
	for _, f := range strings.Fields(s) {
		f = strings.Trim(f, "`.,:;()[]{}\"'")
		f = strings.TrimPrefix(f, "file:")
		if i := strings.IndexByte(f, ':'); i >= 0 {
			f = f[:i] // drop a :line suffix
		}
		if strings.Contains(f, "/") || strings.HasSuffix(f, ".go") {
			out = append(out, f)
		}
	}
	return out
}

// significantWords lowercases s and returns its content words (length ≥ 4, not
// a stopword) as a set — the crude topical fingerprint lessonMatchesTask uses.
func significantWords(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len(w) >= 4 && !lessonStopWords[w] {
			out[w] = true
		}
	}
	return out
}

// lessonStopWords are common tokens that would match almost any lesson and so
// carry no topical signal.
var lessonStopWords = map[string]bool{
	"task": true, "with": true, "this": true, "that": true, "from": true,
	"when": true, "then": true, "than": true, "code": true, "test": true,
	"tests": true, "into": true, "over": true, "your": true, "have": true,
	"here": true, "does": true, "must": true, "only": true, "also": true,
	"they": true, "them": true, "will": true, "each": true, "same": true,
}

func cmdEstimate(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli estimate <task-ref>")
	}
	t, err := store.FindTask(w, f.Pos[0])
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

	// D1 inversion: if this task's agent band (role×model×runtime) has enough
	// history, the empirical distribution IS the estimate and the human PERT
	// above is demoted to the prior. The band comes from the task's own run
	// record; a task never spawned has no band and this stays silent.
	if band, ok := store.TaskBand(w, t.ID); ok {
		var rs []float64
		for _, s := range store.CalibrationSamples(w) {
			if s.Band == band {
				rs = append(rs, s.Ratio())
			}
		}
		if len(rs) >= 10 {
			med, p10, p90 := spm.Median(rs), percentile(rs, 10), percentile(rs, 90)
			te := tp.Expected()
			fmt.Fprintf(ctx.Stdout,
				"  empirical band %s (n=%d): ×%.2f median · p10–p90 ×%.2f–×%.2f hours/point\n"+
					"    → estimate %.1f h (p10–p90 %.1f–%.1f h) — THIS is the estimate; the PERT above is the prior\n"+
					"    (actuals are wall-clock claim→completion, a time proxy until runtimes report token usage)\n",
				band.String(), len(rs), med, p10, p90, med*te, p10*te, p90*te)
		}
	}
	return nil
}

func cmdCriticalPath(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	project := f.Get("project")
	if project == "" && len(f.Pos) > 0 {
		project = f.Pos[0]
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

func cmdWBS(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	project := f.Get("project")
	if project == "" && len(f.Pos) > 0 {
		project = f.Pos[0]
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

func cmdBurndown(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	tasks, err := store.ListTasks(w, f.Get("project"), "")
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

func cmdVelocity(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
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

// cmdCalibrate is the P2 loop's readout: how wrong the estimates actually
// are, measured, not assumed. McConnell's cone becomes YOUR cone.
func cmdCalibrate(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	samples := store.CalibrationSamples(w)
	if len(samples) == 0 {
		fmt.Fprintln(ctx.Stdout, "no history: calibration needs done tasks with both a three-point estimate and claim→completion stamps")
		return nil
	}

	band := func(te float64) string {
		switch {
		case te <= 3:
			return "small (≤3)"
		case te <= 8:
			return "medium (≤8)"
		default:
			return "large (>8)"
		}
	}
	byBand := map[string][]float64{}
	var all []float64
	for _, s := range samples {
		byBand[band(s.Te)] = append(byBand[band(s.Te)], s.Ratio())
		all = append(all, s.Ratio())
	}
	fmt.Fprintln(ctx.Stdout, "by size band:")
	for _, name := range []string{"small (≤3)", "medium (≤8)", "large (>8)"} {
		rs := byBand[name]
		if len(rs) == 0 {
			continue
		}
		// Per-band n-gate: only a band with n>=10 shows a calibrated p10–p90
		// range. A thinner band prints its median marked "provisional" and NO
		// range — a percentile spread over a handful of samples is confidence
		// theater, the exact failure the overall size gate below warns against.
		if len(rs) >= 10 {
			fmt.Fprintf(ctx.Stdout, "%-12s n=%-3d ×%.2f median  p10–p90 ×%.2f–×%.2f hours/point\n",
				name, len(rs), spm.Median(rs), percentile(rs, 10), percentile(rs, 90))
		} else {
			fmt.Fprintf(ctx.Stdout, "%-12s n=%-3d ×%.2f median hours/point  (provisional, n<10 — no calibrated range)\n",
				name, len(rs), spm.Median(rs))
		}
	}
	fmt.Fprintf(ctx.Stdout, "%-12s n=%-3d ×%.2f median hours/point\n", "overall", len(all), spm.Median(all))

	// Agent bands — role × model × runtime. This is the D1 inversion: once a
	// band has n>=10 samples its empirical distribution is the authoritative
	// estimate, not a multiplier beside PERT. Samples with no run record joined
	// (Band.Empty) cannot be attributed to an agent, so they are size-band only.
	byAgent := map[string][]float64{}
	for _, s := range samples {
		if s.Band.Empty() {
			continue
		}
		byAgent[s.Band.String()] = append(byAgent[s.Band.String()], s.Ratio())
	}
	if len(byAgent) > 0 {
		fmt.Fprintln(ctx.Stdout, "\nby agent band (role/model/runtime):")
		names := make([]string, 0, len(byAgent))
		for name := range byAgent {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			rs := byAgent[name]
			// Per-band n-gate: a band with n<10 is provisional and prints its
			// median only — NO p10–p90 range. Printing "×0.03–×0.03" over n=1 as
			// if calibrated is confidence theater; only a band that clears the
			// n>=10 floor earns the range (and the AUTHORITATIVE claim).
			if len(rs) >= 10 {
				fmt.Fprintf(ctx.Stdout, "%-28s n=%-3d ×%.2f median  p10–p90 ×%.2f–×%.2f hours/point  ← AUTHORITATIVE (n≥10: this distribution IS the estimate)\n",
					name, len(rs), spm.Median(rs), percentile(rs, 10), percentile(rs, 90))
			} else {
				fmt.Fprintf(ctx.Stdout, "%-28s n=%-3d ×%.2f median  (provisional, n<10 — no calibrated range)\n",
					name, len(rs), spm.Median(rs))
			}
		}
	} else {
		fmt.Fprintln(ctx.Stdout, "\nby agent band: no done task joins a run record yet (runs predate model-banding, or none recorded)")
	}

	// F1: token-per-point bands. When a band's completing runs used a
	// usage-reporting runtime, output tokens are the REAL unit and wall-clock is
	// demoted to the fallback for runs without usage. This is the caveat every
	// readout above has printed finally coming true: tokens, not a time proxy.
	tokenByAgent := map[string][]float64{}
	tokenSamples := 0
	for _, s := range samples {
		if !s.HasTokens() || s.Band.Empty() {
			continue
		}
		tokenByAgent[s.Band.String()] = append(tokenByAgent[s.Band.String()], s.TokenRatio())
		tokenSamples++
	}
	if len(tokenByAgent) > 0 {
		fmt.Fprintln(ctx.Stdout, "\nby agent band (tokens/point) — PREFERRED (real unit; wall-clock above is the fallback):")
		names := make([]string, 0, len(tokenByAgent))
		for name := range tokenByAgent {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			rs := tokenByAgent[name]
			if len(rs) >= 10 {
				fmt.Fprintf(ctx.Stdout, "%-28s n=%-3d %.0f median tok/point  p10–p90 %.0f–%.0f  ← AUTHORITATIVE (n≥10: tokens ARE the estimate)\n",
					name, len(rs), spm.Median(rs), percentile(rs, 10), percentile(rs, 90))
			} else {
				fmt.Fprintf(ctx.Stdout, "%-28s n=%-3d %.0f median tok/point  (provisional, n<10 — no calibrated range)\n",
					name, len(rs), spm.Median(rs))
			}
		}
	}

	if len(all) < 10 {
		fmt.Fprintf(ctx.Stdout, "insufficient history (n=%d < 10): briefs stay silent — a multiplier from anecdotes is confidence theater\n", len(all))
	} else {
		fmt.Fprintln(ctx.Stdout, "briefs now show the calibrated range beside PERT")
	}
	if tokenSamples > 0 {
		fmt.Fprintf(ctx.Stdout, "(tokens/point is the real unit, from runtime usage on %d sample(s); wall-clock claim→completion is the fallback for runs without usage)\n", tokenSamples)
	} else {
		fmt.Fprintln(ctx.Stdout, "(actuals are wall-clock claim→completion — a time PROXY until runtimes report token usage; opt a runtime in with usage_format: stream-json)")
	}
	return nil
}

// percentile returns the p-th (0..100) percentile of xs by linear
// interpolation on the sorted copy — the p10/p90 spread the calibration
// readout reports so a band's distribution, not just its median, is visible.
// Zero-length returns 0.
func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	if len(s) == 1 {
		return s[0]
	}
	rank := p / 100 * float64(len(s)-1)
	lo := int(rank)
	if lo >= len(s)-1 {
		return s[len(s)-1]
	}
	return s[lo] + (rank-float64(lo))*(s[lo+1]-s[lo])
}

// cmdTaint is the P4 blast-radius query: given a hostile source, which
// briefs consumed it. It does not fix injection — nothing does — but it
// makes the propagation auditable in seconds instead of an unbounded
// suspicion, which is the only honest posture the design claims.
func cmdTaint(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	source := f.Get("origin")
	if source == "" && len(f.Pos) > 0 {
		source = f.Pos[0]
	}
	if source == "" {
		return clikit.Usagef("usage: dacli taint <origin>   (e.g. file:cron/settle.go, external:someuser, or just file: for all)")
	}
	res, err := store.Taint(w, source)
	if err != nil {
		return err
	}
	if len(res.Hits) == 0 {
		fmt.Fprintf(ctx.Stdout, "no artifact carries origin %q — nothing derived from this source\n", source)
		return nil
	}
	for _, h := range res.Hits {
		loc := h.About
		if h.Project != "" {
			loc = h.Project + "/" + h.About
		}
		fmt.Fprintf(ctx.Stdout, "%-6s %-28s by %-14s origin=%s → %s\n", h.Kind, h.ID, h.Actor, h.Origin, loc)
	}
	exposed := res.ExposedBriefs(w)
	sort.Strings(exposed)
	scope := fmt.Sprintf("%d project(s)", len(res.Projects))
	if res.TreeWide {
		scope = "TREE-WIDE (a workspace-scoped hit reaches every project's briefs)"
	}
	fmt.Fprintf(ctx.Stdout, "\nblast radius: %d artifact(s), %s, %d brief(s) exposed\n",
		len(res.Hits), scope, len(exposed))
	if len(exposed) > 0 {
		fmt.Fprintf(ctx.Stdout, "exposed briefs: %s\n", strings.Join(exposed, ", "))
	}
	// Reviewer F4: origin is self-reported, so this is a floor. An artifact
	// whose author omitted --origin carries "agent" and is invisible here.
	fmt.Fprintln(ctx.Stdout, "this is a LOWER BOUND: only honestly-labeled provenance is traced — unlabeled artifacts are invisible.")
	fmt.Fprintln(ctx.Stdout, "(an audit, not a fix — review these briefs' consumers; injection prevention is unsolved, RUNTIMES § 18)")
	return nil
}

// cmdDoctor runs anti-pattern detectors over tasks, risks, and the event
// log. Informational: the point is visibility while the pattern is cheap.
func cmdDoctor(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	found := 0
	report := func(pattern, detail string) {
		found++
		fmt.Fprintf(ctx.Stdout, "%-22s %s\n", pattern+":", detail)
	}

	tasks, _ := store.ListTasks(w, "", "")
	var mustsOpen, done, active int
	var lowerActive []string
	var brokenSpans []string
	for _, t := range tasks {
		switch t.Status {
		case model.StatusDone:
			done++
			// Data-integrity: a done task claimed (E3 spawn stamp) but never
			// stamped "completed by" has a broken calibration span — it can never
			// produce a sample. Name it so the drift that E1/E7 fixed can't hide.
			if store.LogHasStamp(t, "claimed by") && !store.LogHasStamp(t, "completed by") {
				brokenSpans = append(brokenSpans, fmt.Sprintf("%03d-%s", t.Seq, t.Slug))
			}
		case model.StatusActive:
			active++
			if model.Priority(t.Priority()).Rank() > 0 {
				lowerActive = append(lowerActive, fmt.Sprintf("%03d-%s(%s)", t.Seq, t.Slug, clikit.OrDash(t.Priority())))
			}
		case model.StatusOpen:
			if model.Priority(t.Priority()).Rank() == 0 && t.Priority() != "" {
				mustsOpen++
			}
		}
	}

	if mustsOpen > 0 && len(lowerActive) > 0 {
		report("cart-before-the-horse", fmt.Sprintf("%d must task(s) sit open while lower-priority work is active: %s",
			mustsOpen, strings.Join(lowerActive, ", ")))
	}
	if active >= 3 && done == 0 {
		report("burning-across", fmt.Sprintf("%d tasks active, 0 done — finish before starting; redirect free agents to help", active))
	}
	if len(brokenSpans) > 0 {
		report("broken-calibration-span", fmt.Sprintf("%d done task(s) claimed but never stamped 'completed by' — calibration cannot size them: %s",
			len(brokenSpans), strings.Join(brokenSpans, ", ")))
	}
	// Data-integrity: a task file living in more than one status folder is the
	// duplicate-task drift that made FindTask fail with "ambiguous" on the same
	// task twice (026 lived in both open/ and done/). ListTasks now dedups it
	// away; name the paths so the drift stays visible instead of silent.
	if dups, _ := store.DuplicateTaskFiles(w); len(dups) > 0 {
		for _, d := range dups {
			report("duplicate-task-file", fmt.Sprintf("%03d-%s exists in %d status folders: %s",
				d.Seq, d.Slug, len(d.Paths), strings.Join(d.Paths, ", ")))
		}
	}

	findings, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventFinding}})
	noteFindings := 0
	if ps, _ := store.ListProjects(w); ps != nil {
		for _, p := range ps {
			ns, _ := store.ListNotes(w, p.Slug, model.NoteFinding)
			noteFindings += len(ns)
			risks, _ := store.ListRisks(w, p.Slug)
			for _, r := range risks {
				if r.Rank() == 1 && strings.TrimSpace(r.Action) == "" {
					report("unmanaged-risk", fmt.Sprintf("%s/%s is rank 1 with no action plan", p.Slug, r.Slug))
				}
			}
		}
	}
	if len(findings)+noteFindings >= 5 && done == 0 {
		report("analysis-paralysis", fmt.Sprintf("%d findings recorded, 0 tasks done — deliver something", len(findings)+noteFindings))
	}
	if qs, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}, Pending: true}); len(qs) > 0 {
		report("unanswered-questions", fmt.Sprintf("%d question(s) open — the asking tasks are blocked until someone answers", len(qs)))
	}
	for _, r := range func() []team.Role { rs, _ := store.LoadRoles(w); return rs }() {
		if r.WIP > 0 {
			if n := store.ActiveInRole(w, r.Name); n > r.WIP {
				report("wip-exceeded", fmt.Sprintf("role %s has %d active agents against a limit of %d", r.Name, n, r.WIP))
			}
		}
	}

	if found == 0 {
		fmt.Fprintln(ctx.Stdout, "no anti-patterns detected")
	}
	return nil
}

// cmdStandup is derived entirely from the log and the tasks — no agent ever
// files a status report.
func cmdStandup(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	tasks, _ := store.ListTasks(w, "", "")
	events, _ := eventlog.List(w, eventlog.Query{})

	type roll struct {
		doing, doneT, blocked []string
		events                int
	}
	rolls := map[string]*roll{}
	get := func(id string) *roll {
		if rolls[id] == nil {
			rolls[id] = &roll{}
		}
		return rolls[id]
	}
	for _, t := range tasks {
		if t.Owner() == "" {
			continue
		}
		label := fmt.Sprintf("%03d-%s", t.Seq, t.Slug)
		switch t.Status {
		case model.StatusActive:
			get(t.Owner()).doing = append(get(t.Owner()).doing, label)
		case model.StatusDone:
			get(t.Owner()).doneT = append(get(t.Owner()).doneT, label)
		case model.StatusBlocked:
			get(t.Owner()).blocked = append(get(t.Owner()).blocked, label)
		}
	}
	for _, e := range events {
		get(e.Actor).events++
	}

	ids := make([]string, 0, len(rolls))
	for id := range rolls {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		r := rolls[id]
		fmt.Fprintf(ctx.Stdout, "%s (%d events)\n", id, r.events)
		if len(r.doneT) > 0 {
			fmt.Fprintf(ctx.Stdout, "  done:        %s\n", strings.Join(r.doneT, ", "))
		}
		if len(r.doing) > 0 {
			fmt.Fprintf(ctx.Stdout, "  doing:       %s\n", strings.Join(r.doing, ", "))
		}
		if len(r.blocked) > 0 {
			fmt.Fprintf(ctx.Stdout, "  impediments: %s\n", strings.Join(r.blocked, ", "))
		}
	}
	return nil
}
