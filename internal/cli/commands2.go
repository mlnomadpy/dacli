// Second slice of v0.1: sync, next (the CPM scheduler on real tasks), risks,
// glossary, block, and --json on the read paths.
package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
)

func emitJSON(ctx *Ctx, v any) error {
	enc := json.NewEncoder(ctx.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func cmdSync(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	res, err := eventlog.Sync(w, id.ID, id.CanMutate)
	if err != nil {
		return err
	}
	for _, n := range res.Notes {
		fmt.Fprintf(ctx.Stdout, "applied %s\n", n)
	}
	fmt.Fprintf(ctx.Stdout, "sync: %d applied, %d left pending\n", res.Applied, res.Skipped)
	return nil
}

func cmdProjectShow(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli project show <slug>")
	}
	p, err := store.LoadProject(w, f.pos[0])
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, mdstore.Render(p.Doc))
	return nil
}

func cmdTaskBlock(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli task block <ref> [--by ref] [--why text]")
	}
	t, err := store.FindTask(w, f.pos[0])
	if err != nil {
		return err
	}
	why := f.get("why")
	if by := f.get("by"); by != "" {
		why = "blocked_by [[" + by + "]] " + why
	}
	if !id.CanMutate(t.Owner()) {
		if _, err := eventlog.Append(w, id.ID, model.EventBlock, t.ID, "", why); err != nil {
			return err
		}
		fmt.Fprintln(ctx.Stdout, "block recorded as event (read-only grant)")
		return nil
	}
	if by := f.get("by"); by != "" {
		t.Doc.Front.Set("blocked_by", "[["+by+"]]")
	}
	store.AppendLog(t, "blocked: "+why)
	if err := store.SaveTask(t); err != nil {
		return err
	}
	if err := store.MoveTask(w, t, model.StatusBlocked); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "blocked: %03d-%s\n", t.Seq, t.Slug)
	return nil
}

func cmdRiskAdd(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 || f.get("project") == "" || f.get("impact") == "" || f.get("likelihood") == "" {
		return usagef("usage: dacli risk add <title> --project <slug> --impact high|medium|low --likelihood high|medium|low [--indicator text]... [--action text]")
	}
	r, err := store.CreateRisk(w, id.ID, f.get("project"), strings.Join(f.pos, " "),
		model.Level(f.get("impact")), model.Level(f.get("likelihood")),
		f.all("indicator"), f.get("action"))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "risk %s recorded (rank %d)\n", r.Slug, r.Rank())
	if r.Rank() <= 2 && strings.TrimSpace(r.Action) == "" {
		fmt.Fprintf(ctx.Stderr, "warning: rank-%d risk with no action plan — ranks 1 and 2 require one\n", r.Rank())
	}
	return nil
}

func cmdRiskList(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	project := f.get("project")
	if project == "" && len(f.pos) > 0 {
		project = f.pos[0]
	}
	if project == "" {
		return usagef("usage: dacli risk list <project>")
	}
	risks, err := store.ListRisks(w, project)
	if err != nil {
		return err
	}
	for _, r := range risks {
		flag := ""
		if r.Rank() <= 2 && strings.TrimSpace(r.Action) == "" {
			flag = "  ⚠ no action plan"
		}
		fmt.Fprintf(ctx.Stdout, "rank %d  %-8s×%-8s %s%s\n", r.Rank(), r.Impact, r.Likelihood, r.Title, flag)
	}
	return nil
}

func cmdGlossary(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli glossary <project> [--term t --def text]")
	}
	project := f.pos[0]
	if term := f.get("term"); term != "" {
		if f.get("def") == "" {
			return usagef("--term requires --def")
		}
		if err := store.GlossaryAdd(w, id.ID, project, term, f.get("def")); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "defined %q\n", term)
		return nil
	}
	fmt.Fprint(ctx.Stdout, store.GlossaryRead(w, project))
	return nil
}

// cmdNext is the scheduling answer: MoSCoW first, then critical path. For a
// human team CPM says where delay hurts; for a parent agent it says which
// tasks to spawn children on FIRST — fanning out onto slack tasks while the
// critical path idles is the default agent failure.
func cmdNext(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	limit := 3
	if n := f.get("parallel"); n != "" {
		fmt.Sscanf(n, "%d", &limit)
	}

	tasks, err := store.ListTasks(w, f.get("project"), "")
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

	// ready: every non-SS dependency is done. SS permits overlap — that is
	// the entire reason dependency types exist.
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
	// are missing, and SAY SO — a priority-sorted list must not masquerade
	// as a critical path.
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
				typ := spm.DepType(d.Type)
				edges = append(edges, spm.Edge{From: dep.ID, To: t.ID, Type: typ})
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
	// A blocked higher-priority task is worth announcing rather than
	// silently skipping past.
	for _, t := range open {
		if !ready(t) && model.Priority(t.Priority()).Rank() < top {
			fmt.Fprintf(ctx.Stderr, "note: %03d-%s (%s) outranks these but is waiting on a dependency\n", t.Seq, t.Slug, t.Priority())
		}
	}
	return nil
}

// --- JSON variants: stable shapes are the API (ARCHITECTURE § 4) ---

type taskJSON struct {
	ID       string `json:"id"`
	Seq      int    `json:"seq"`
	Slug     string `json:"slug"`
	Project  string `json:"project"`
	Status   string `json:"status"`
	Priority string `json:"priority,omitempty"`
	Title    string `json:"title"`
	Done     int    `json:"acceptance_done"`
	Total    int    `json:"acceptance_total"`
}

func taskToJSON(t *store.Task) taskJSON {
	boxes := t.Acceptance()
	done := 0
	for _, b := range boxes {
		if b.Done {
			done++
		}
	}
	return taskJSON{ID: t.ID, Seq: t.Seq, Slug: t.Slug, Project: t.Project,
		Status: string(t.Status), Priority: t.Priority(), Title: t.Title,
		Done: done, Total: len(boxes)}
}

func cmdTaskListJSON(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	ts, err := store.ListTasks(w, f.get("project"), model.Status(f.get("status")))
	if err != nil {
		return err
	}
	out := []taskJSON{}
	for _, t := range ts {
		out = append(out, taskToJSON(t))
	}
	return emitJSON(ctx, out)
}

func cmdContextJSON(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli context <task-ref> [--budget N]")
	}
	budget := 0
	fmt.Sscanf(f.get("budget"), "%d", &budget)
	b, err := brief.Assemble(w, f.pos[0], brief.Options{Budget: budget})
	if err != nil {
		return err
	}
	type section struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	out := struct {
		TaskID   string    `json:"task_id"`
		Sections []section `json:"sections"`
		Omitted  []string  `json:"omitted"`
	}{TaskID: b.TaskID, Omitted: b.Omitted}
	if out.Omitted == nil {
		out.Omitted = []string{}
	}
	for _, s := range b.Sections {
		out.Sections = append(out.Sections, section{s.Title, s.Content})
	}
	return emitJSON(ctx, out)
}
