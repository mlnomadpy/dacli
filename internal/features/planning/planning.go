// Package planning is the work-definition slice: projects, tasks, risks,
// and the glossary — the objects every other slice reads.
package planning

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "project add", Brief: "Create a project", Run: cmdProjectAdd},
	{Path: "project list", Brief: "List projects", Run: cmdProjectList},
	{Path: "project show", Brief: "Show a project", Run: cmdProjectShow},
	{Path: "task add", Brief: "Create a task", Run: cmdTaskAdd},
	{Path: "task list", Brief: "List tasks, optionally by status", Run: cmdTaskList},
	{Path: "task show", Brief: "Show a task", Run: cmdTaskShow},
	{Path: "task claim", Brief: "Take ownership of a task", Run: cmdTaskClaim},
	{Path: "task check", Brief: "Check acceptance boxes (--n N or --all)", Run: cmdTaskCheck},
	{Path: "task done", Brief: "Move a task to done; verifies acceptance, refuses if unmet", Run: cmdTaskDone},
	{Path: "task block", Brief: "Mark a task blocked", Run: cmdTaskBlock},
	{Path: "risk add", Brief: "Record a risk in the impact x likelihood matrix", Run: cmdRiskAdd},
	{Path: "risk list", Brief: "List risks by rank; rank 1 and 2 require an action plan", Run: cmdRiskList},
	{Path: "glossary", Brief: "Show or edit the project term list", Run: cmdGlossary},
}

func cmdProjectAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli project add <title> [--slug s] [--goal g] [--stage definition|elicitation|approach|design]")
	}
	title := strings.Join(f.Pos, " ")
	p, err := store.CreateProject(w, id.ID, title, f.Get("slug"), f.Get("goal"), f.Get("stage"))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "project %s created (stage: %s)\n", p.Slug, p.Stage)

	// --template attaches controlled steps at birth. Solo is the default by
	// absence: no template means no gates, per TEMPLATES.md § 2. When the flag
	// is omitted, fall back to the workspace default `dacli init --template`
	// recorded, so init's seeding actually reaches the first project.
	tmpl := f.Get("template")
	if tmpl == "" {
		tmpl = w.DefaultTemplate
	}
	if tmpl != "" && tmpl != "solo" {
		first, err := gates.Attach(w, p.Slug, tmpl)
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "template %s attached (stage: %s)\n", tmpl, first.Name)
	}
	return nil
}

func cmdProjectList(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	ps, err := store.ListProjects(w)
	if err != nil {
		return err
	}
	for _, p := range ps {
		fmt.Fprintf(ctx.Stdout, "%-16s %-12s %s\n", p.Slug, p.Stage, p.Title)
	}
	return nil
}

func cmdProjectShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli project show <slug>")
	}
	p, err := store.LoadProject(w, f.Pos[0])
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, mdstore.Render(p.Doc))
	return nil
}

func cmdTaskAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 || f.Get("project") == "" {
		return clikit.Usagef("usage: dacli task add <title> --project <slug> [--priority must|should|could|wont] [--estimate o,m,p] [--accept criterion]... [--so-that why] [--parent ref] [--depends-on ref[:TYPE]]... [--force]")
	}
	title := strings.Join(f.Pos, " ")

	// Ambiguity lint on the title, before the task exists. Titles get the
	// strict pass because a vague title becomes three different deliverables.
	if finds := spm.Scan(title, spm.Options{}); len(finds) > 0 {
		fmt.Fprintf(ctx.Stderr, "warning: ambiguous title —\n")
		for _, fd := range finds {
			fmt.Fprintf(ctx.Stderr, "  %s\n", fd)
		}
	}

	// Near-duplicate dedup against the open backlog, before the task exists.
	// A review auditor that re-discovers an issue a prior cycle already
	// queued must be told "already filed", not left to manufacture a second
	// task for the same work (dacli task 116).
	if !f.Bool("force") {
		if dup, score, err := store.FindNearDuplicateTask(w, f.Get("project"), title); err != nil {
			return err
		} else if dup != nil {
			return clikit.Refusedf("title is a %.0f%% near-duplicate of open task %03d-%s (%q) — check it before filing, or re-run with --force to file anyway", score*100, dup.Seq, dup.Slug, dup.Title)
		}
	}

	t, err := store.CreateTask(w, id.ID, f.Get("project"), title, store.TaskOpts{
		Priority:  f.Get("priority"),
		Estimate:  f.Get("estimate"),
		Accept:    f.All("accept"),
		SoThat:    f.Get("so-that"),
		Context:   f.Get("context"),
		DependsOn: f.All("depends-on"),
		Parent:    f.Get("parent"),
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "task %03d-%s created (%s)\n", t.Seq, t.Slug, t.ID)
	if len(t.Acceptance()) == 0 {
		fmt.Fprintf(ctx.Stderr, "warning: no acceptance criteria — an agent given this task cannot know when to stop (add --accept)\n")
	}
	return nil
}

func cmdTaskList(ctx *clikit.Ctx, args []string) error {
	if ctx.JSON {
		return cmdTaskListJSON(ctx, args)
	}
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	ts, err := store.ListTasks(w, f.Get("project"), model.Status(f.Get("status")))
	if err != nil {
		return err
	}
	for _, t := range ts {
		boxes := t.Acceptance()
		done := 0
		for _, b := range boxes {
			if b.Done {
				done++
			}
		}
		fmt.Fprintf(ctx.Stdout, "%-10s %03d-%-28s %-8s %-7s [%d/%d] %s\n",
			t.Project, t.Seq, t.Slug, t.Status, t.Priority(), done, len(boxes), t.Title)
	}
	return nil
}

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

func cmdTaskListJSON(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	ts, err := store.ListTasks(w, f.Get("project"), model.Status(f.Get("status")))
	if err != nil {
		return err
	}
	out := []taskJSON{}
	for _, t := range ts {
		boxes := t.Acceptance()
		done := 0
		for _, b := range boxes {
			if b.Done {
				done++
			}
		}
		out = append(out, taskJSON{ID: t.ID, Seq: t.Seq, Slug: t.Slug, Project: t.Project,
			Status: string(t.Status), Priority: t.Priority(), Title: t.Title,
			Done: done, Total: len(boxes)})
	}
	return clikit.EmitJSON(ctx, out)
}

func cmdTaskShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli task show <ref>")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, mdstore.Render(t.Doc))
	return nil
}

func cmdTaskClaim(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli task claim <ref>")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}

	if owner := t.Owner(); owner != "" && owner != id.ID && t.Status == model.StatusActive {
		return clikit.Refusedf("task is owned by %s and active; ask them or wait for release", owner)
	}
	if !id.CanMutate(t.Owner()) {
		// A read-only agent claims via an event, not a rewrite.
		if _, err := eventlog.Append(w, id.ID, model.EventClaim, t.ID, "", ""); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "claim recorded as event (%s); the owner applies it on sync\n", id.MutateRefusal())
		return nil
	}

	t.Doc.Front.Set("owner", id.ID)
	store.AppendLog(t, "claimed by "+id.ID)
	if err := store.SaveTask(t); err != nil {
		return err
	}
	if t.Status == model.StatusOpen {
		if err := store.MoveTask(w, t, model.StatusActive); err != nil {
			return err
		}
	}
	fmt.Fprintf(ctx.Stdout, "claimed %03d-%s\n", t.Seq, t.Slug)
	return nil
}

func cmdTaskCheck(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli task check <ref> [--n N | --all]")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	if !id.CanMutate(t.Owner()) {
		return clikit.Refusedf("only the owner (%s) checks acceptance boxes; report a finding instead", t.Owner())
	}
	sec, ok := t.Doc.Section("Acceptance")
	if !ok {
		return fmt.Errorf("task has no acceptance section")
	}
	boxes := mdstore.Checkboxes(sec.Content)
	if f.Bool("all") {
		for i := range boxes {
			boxes[i].Done = true
		}
	} else {
		n, err := strconv.Atoi(f.Get("n"))
		if err != nil || n < 1 || n > len(boxes) {
			return clikit.Usagef("--n must be 1..%d", len(boxes))
		}
		boxes[n-1].Done = true
	}
	t.Doc.SetSection("Acceptance", mdstore.RenderCheckboxes(boxes))
	return store.SaveTask(t)
}

func cmdTaskDone(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli task done <ref>")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	if !id.CanMutate(t.Owner()) {
		if _, err := eventlog.Append(w, id.ID, model.EventProposeStatus, t.ID, "", "propose: done"); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "done proposed as event (%s)\n", id.MutateRefusal())
		return nil
	}

	// done VERIFIES, not just records: unchecked acceptance is a refusal
	// naming the criterion — "no" is an answer, not a failure.
	var unmet []string
	for _, box := range t.Acceptance() {
		if !box.Done {
			unmet = append(unmet, box.Text)
		}
	}
	if len(unmet) > 0 {
		return clikit.Refusedf("acceptance unmet:\n  - %s\nfix, `task check`, or `ask` if the criterion is wrong — do not retry", strings.Join(unmet, "\n  - "))
	}

	// One canonical close: CloseTask stamps "completed by" (the actuals capture
	// field) and moves to done, the same primitive `accept` uses.
	if err := store.CloseTask(w, t, id.ID); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "done: %03d-%s\n", t.Seq, t.Slug)
	return nil
}

func cmdTaskBlock(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli task block <ref> [--by ref] [--why text]")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	why := f.Get("why")
	if by := f.Get("by"); by != "" {
		why = "blocked_by [[" + by + "]] " + why
	}
	if !id.CanMutate(t.Owner()) {
		if _, err := eventlog.Append(w, id.ID, model.EventBlock, t.ID, "", why); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "block recorded as event (%s)\n", id.MutateRefusal())
		return nil
	}
	if by := f.Get("by"); by != "" {
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

func cmdRiskAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 || f.Get("project") == "" || f.Get("impact") == "" || f.Get("likelihood") == "" {
		return clikit.Usagef("usage: dacli risk add <title> --project <slug> --impact high|medium|low --likelihood high|medium|low [--indicator text]... [--action text]")
	}
	r, err := store.CreateRisk(w, id.ID, f.Get("project"), strings.Join(f.Pos, " "),
		model.Level(f.Get("impact")), model.Level(f.Get("likelihood")),
		f.All("indicator"), f.Get("action"))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "risk %s recorded (rank %d)\n", r.Slug, r.Rank())
	if r.Rank() <= 2 && strings.TrimSpace(r.Action) == "" {
		fmt.Fprintf(ctx.Stderr, "warning: rank-%d risk with no action plan — ranks 1 and 2 require one\n", r.Rank())
	}
	return nil
}

func cmdRiskList(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	project := f.Get("project")
	if project == "" && len(f.Pos) > 0 {
		project = f.Pos[0]
	}
	if project == "" {
		return clikit.Usagef("usage: dacli risk list <project>")
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

func cmdGlossary(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli glossary <project> [--term t --def text]")
	}
	project := f.Pos[0]
	if term := f.Get("term"); term != "" {
		if f.Get("def") == "" {
			return clikit.Usagef("--term requires --def")
		}
		if err := store.GlossaryAdd(w, id.ID, project, term, f.Get("def")); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "defined %q\n", term)
		return nil
	}
	fmt.Fprint(ctx.Stdout, store.GlossaryRead(w, project))
	return nil
}
