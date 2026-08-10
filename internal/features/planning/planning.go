// Package planning is the work-definition slice: projects, tasks, risks,
// and the glossary — the objects every other slice reads.
package planning

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "project add", Brief: "Create a project", Mutates: true, Run: cmdProjectAdd},
	{Path: "project list", Brief: "List projects", Run: cmdProjectList},
	{Path: "project show", Brief: "Show a project", Run: cmdProjectShow},
	{Path: "project rm", Brief: "Delete a project and everything filed under it (irreversible; requires --force)", Mutates: true, Run: cmdProjectRm},
	{Path: "task add", Brief: "Create a task", Run: cmdTaskAdd},
	{Path: "task list", Brief: "List tasks, optionally by status", JSON: true, Run: cmdTaskList},
	{Path: "task show", Brief: "Show a task", Run: cmdTaskShow},
	{Path: "task claim", Brief: "Take ownership of a task", Run: cmdTaskClaim},
	{Path: "task check", Brief: "Check acceptance boxes (--n N or --all)", Run: cmdTaskCheck},
	{Path: "task done", Brief: "Move a task to done; verifies acceptance, refuses if unmet", Run: cmdTaskDone},
	{Path: "task block", Brief: "Mark a task blocked", Run: cmdTaskBlock},
	{Path: "task reopen", Brief: "Reopen a wrongly-closed task, clearing its acceptance boxes (--reason required)", Mutates: true, Usage: "dacli task reopen <ref> --reason \"<what makes the close wrong>\"", Run: cmdTaskReopen},
	{Path: "task rm", Brief: "Remove a task that should never have existed; refuses while anything references it, and refuses a done task without --force", Mutates: true, Usage: "dacli task rm <ref> [--force]", Run: cmdTaskRm},
	{Path: "task estimate", Brief: "Size an existing task: --estimate o,m,p (three-point; a scalar hides the risk). Sizing the backlog is what makes critical-path and `next --parallel` work", Run: cmdTaskEstimate},
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject("goal", "slug", "stage", "template"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli project add <title> [--slug s] [--goal g] [--stage definition|elicitation|approach|design]")
	}
	if err := clikit.RequireRW(id, "creating a project"); err != nil {
		return err
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
	// This command takes no flags, so ANY flag is a typo. An empty allowlist
	// rejects every one — without it a mistyped flag was dropped and the
	// command ran as if nothing were wrong.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject(); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
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

// cmdProjectRm is the recovery path for a project created by mistake (e.g. an
// `adopt` that guessed the wrong slug, see task 118): it deletes the project
// directory and everything filed under it — tasks, notes, risks, glossary.
// That is irreversible, so it refuses without --force and reports the size of
// the blast radius (task count) so the caller isn't guessing at it.
func cmdProjectRm(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("force"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli project rm <slug> --force")
	}
	slug := f.Pos[0]
	if !id.CanMutate("") {
		return clikit.Refusedf("%s cannot delete a project", id.MutateRefusal())
	}
	p, err := store.LoadProject(w, slug)
	if err != nil {
		return err
	}
	tasks, err := store.ListTasks(w, slug, "")
	if err != nil {
		return err
	}
	if !f.Bool("force") {
		return clikit.Refusedf("project %s (%q) has %d task(s) filed under it — rm deletes the whole project directory (tasks, notes, risks, glossary), irreversibly; re-run with --force to confirm", p.Slug, p.Title, len(tasks))
	}
	if err := store.DeleteProject(w, slug); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "project %s deleted (%d task(s) removed with it)\n", slug, len(tasks))
	return nil
}

func cmdTaskAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("project", "force", "priority", "estimate", "accept", "so-that", "context", "depends-on", "parent"); err != nil {
		return err
	}
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
	// Both the text and JSON list paths need this: a dropped --status filter
	// silently lists EVERY task, which reads as a correct answer to a question
	// nobody asked. The 175 sweep missed both, and an invariant test over the
	// exit-code contract is what found them (dacli 202).
	if err := f.Reject("project", "status"); err != nil {
		return err
	}
	status, serr := model.ParseStatus(f.Get("status"))
	if serr != nil {
		return clikit.Usagef("%v", serr)
	}
	ts, err := store.ListTasks(w, f.Get("project"), status)
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
	// Both the text and JSON list paths need this: a dropped --status filter
	// silently lists EVERY task, which reads as a correct answer to a question
	// nobody asked. The 175 sweep missed both, and an invariant test over the
	// exit-code contract is what found them (dacli 202).
	if err := f.Reject("project", "status"); err != nil {
		return err
	}
	status, serr := model.ParseStatus(f.Get("status"))
	if serr != nil {
		return clikit.Usagef("%v", serr)
	}
	ts, err := store.ListTasks(w, f.Get("project"), status)
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
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
	if err := f.Reject("n", "all"); err != nil {
		return err
	}
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
	if err := f.Reject("allow-unverified"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli task done <ref> [--allow-unverified]")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	allowUnverified := f.Bool("allow-unverified")

	// A task with no acceptance criteria has nothing to verify: the unmet-box
	// scan below finds an empty list and would pass, so zero boxes read as all
	// boxes and `done` — which "VERIFIES, not just records" — would certify
	// nothing (dacli 289). Refuse before the propose branch so a read-only agent
	// cannot even propose closing an unverifiable task; the owner opts into an
	// explicitly unverified close with --allow-unverified, the same rule the
	// accept and propose→sync paths enforce.
	if !store.HasAcceptanceCriteria(t) && !allowUnverified {
		return clikit.Refusedf("task %03d has no acceptance criteria: closing it would verify nothing. Add at least one criterion (edit the task, or refile with `task add --accept`), or pass --allow-unverified to close it as explicitly UNVERIFIED — do not retry", t.Seq)
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

	// Record an unverified close on the task itself so the trajectory never
	// implies a verification that could not have happened (dacli 289).
	if !store.HasAcceptanceCriteria(t) {
		store.AppendLog(t, "closed with NO acceptance criteria — UNVERIFIED (--allow-unverified)")
	}

	// One canonical close: CloseTask stamps "completed by" (the actuals capture
	// field) and moves to done, the same primitive `accept` uses.
	if err := store.CloseTask(w, t, id.ID); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "done: %03d-%s\n", t.Seq, t.Slug)
	return nil
}

// cmdTaskEstimate sizes an existing task. Estimates could only be set at
// creation, so any backlog filed without them was permanently unsizable — and
// every scheduling command that depends on estimates (critical-path, next's
// slack ordering, `next --parallel`) silently fell back to MoSCoW-then-sequence,
// the one ordering that cannot tell you what runs concurrently. Sizing after
// the fact is the normal case: you learn the shape of work by looking at it
// (dacli 228).
func cmdTaskEstimate(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("estimate", "force"); err != nil {
		return err
	}
	est := f.Get("estimate")
	if len(f.Pos) == 0 || est == "" {
		return clikit.Usagef("usage: dacli task estimate <ref> --estimate o,m,p [--force]   (optimistic,probable,pessimistic)")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	// Sizing rewrites the task, so it obeys the same ownership rule every other
	// mutation does rather than inventing a looser one. --force mirrors
	// `accept --force`: root adopts a task orphaned by a finished agent, which
	// is the only way a backlog left behind by a dead loop can be sized at all.
	if !id.CanMutate(t.Owner()) {
		if id.ID != agentid.RootID || !f.Bool("force") {
			return clikit.Refusedf("cannot size %03d-%s: %s (root can adopt it with --force)", t.Seq, t.Slug, id.MutateRefusal())
		}
		prev := t.Owner()
		t.Doc.Front.Set("owner", id.ID)
		store.AppendLog(t, fmt.Sprintf("adopted by %s (owner %s orphaned)", id.ID, clikit.OrDash(prev)))
	}
	if err := store.SetEstimate(w, t, est); err != nil {
		return clikit.Usagef("%v", err)
	}
	tp, ok := t.Estimate()
	if !ok {
		return fmt.Errorf("estimate written but did not parse back — refusing to report a size that will not survive a reload")
	}
	fmt.Fprintf(ctx.Stdout, "sized %03d-%s: Te %.1f (o %g, m %g, p %g)\n", t.Seq, t.Slug, tp.Expected(), tp.Optimistic, tp.Probable, tp.Pessimistic)
	return nil
}

func cmdTaskBlock(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject("by", "why"); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject("action", "impact", "indicator", "likelihood", "project"); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject("project"); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject("def", "term"); err != nil {
		return err
	}
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

// cmdTaskReopen moves a wrongly-closed task back to open.
//
// Closing was a one-way door: a task force-accepted by mistake could only be
// corrected by editing the markdown store by hand, which is what happened to
// tasks 336 and 339 when `accept --force` was run over a batch nobody read
// (dacli 340). The tool's product is a record, and it gave no command to fix
// the one thing it exists to keep.
func cmdTaskReopen(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("reason"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli task reopen <ref> --reason \"<what makes the close wrong>\"")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	if !id.CanMutate(t.Owner()) {
		return clikit.Refusedf("%03d-%s is owned by %s — only its owner or root can reopen it", t.Seq, t.Slug, clikit.OrDash(t.Owner()))
	}
	reason := f.Get("reason")
	if strings.TrimSpace(reason) == "" {
		return clikit.Usagef("dacli task reopen needs --reason: a reopen with no reason is a mystery to the next reader")
	}
	cleared, err := store.ReopenTask(w, t, id.ID, reason)
	if err != nil {
		return clikit.Usagef("%v", err)
	}
	// Say what was cleared. Silently unchecking boxes would replace one false
	// record with a different one.
	fmt.Fprintf(ctx.Stdout, "reopened %03d-%s — cleared %d acceptance box(es); the close claimed work that was not verified\n", t.Seq, t.Slug, cleared)
	return nil
}

// cmdTaskRm deletes a task that should never have existed.
//
// For a probe, a duplicate or a mis-filed entry — NOT for retracting real work,
// which is corrected by reopening so the record stays visible. It refuses while
// anything still points at the task, because a dangling reference fails far
// from the deletion that caused it and names neither.
func cmdTaskRm(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("force"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli task rm <ref> [--force]")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	if !id.CanMutate(t.Owner()) {
		return clikit.Refusedf("%03d-%s is owned by %s — only its owner or root can remove it", t.Seq, t.Slug, clikit.OrDash(t.Owner()))
	}
	// A task with a Log has history, and history is corrected by reopening, not
	// by deletion. --force is the deliberate override for the case where the
	// history is itself the mistake.
	if t.Status == model.StatusDone && !f.Bool("force") {
		return clikit.Refusedf("%03d-%s is DONE — removing it erases the record of work that happened. Use `dacli task reopen` to correct a wrong close, or pass --force if this task should never have existed", t.Seq, t.Slug)
	}
	if err := store.RemoveTask(w, t); err != nil {
		var ref store.ErrReferenced
		if errors.As(err, &ref) {
			return clikit.Refusedf("%v — remove the reference first, or reopen the task instead of deleting it", ref)
		}
		return err
	}
	fmt.Fprintf(ctx.Stdout, "removed %03d-%s\n", t.Seq, t.Slug)
	return nil
}
