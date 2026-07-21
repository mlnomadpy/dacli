// Fifth slice: roles wired to spawn (WIP enforced), team views, the doctor,
// standup, and retro.
package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

func cmdRoleAdd(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli role add <name> [--summary s] [--skill s]... [--scope glob]... [--out-of-scope glob]... [--shortcut n]... [--escalate-to role]... [--grant ro|rw] [--wip N]")
	}
	r := team.Role{
		Name:       f.pos[0],
		Summary:    f.get("summary"),
		Skills:     f.all("skill"),
		Scope:      f.all("scope"),
		OutOfScope: f.all("out-of-scope"),
		Shortcuts:  f.all("shortcut"),
		EscalateTo: f.all("escalate-to"),
		Grant:      f.get("grant"),
		Runtime:    f.get("runtime"),
		Model:      f.get("model"),
	}
	fmt.Sscanf(f.get("wip"), "%d", &r.WIP)
	fmt.Sscanf(f.get("max-points"), "%g", &r.MaxPoints)

	// A role must change what an agent can do, not just what it calls
	// itself. A name-only role is cosplay; warn, don't refuse — the fields
	// can be added later, but the warning should sting now.
	if len(r.Skills)+len(r.Scope)+len(r.Shortcuts)+len(r.EscalateTo) == 0 && r.Grant == "" && r.WIP == 0 && r.Model == "" && r.Runtime == "" && r.MaxPoints == 0 {
		fmt.Fprintln(ctx.Stderr, "warning: this role changes nothing mechanical (no skills, scope, shortcuts, escalation, grant, or wip) — it is a costume, not a role")
	}
	if err := store.CreateRole(w, id.ID, r); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "role %s defined\n", r.Name)
	return nil
}

func cmdRoleList(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	roles, _ := store.LoadRoles(w)
	for _, r := range roles {
		extras := []string{}
		if len(r.Skills) > 0 {
			extras = append(extras, fmt.Sprintf("skills:%d", len(r.Skills)))
		}
		if len(r.Scope) > 0 {
			extras = append(extras, fmt.Sprintf("scope:%d", len(r.Scope)))
		}
		if r.WIP > 0 {
			extras = append(extras, fmt.Sprintf("wip:%d", r.WIP))
		}
		if r.Model != "" {
			extras = append(extras, "model:"+r.Model)
		}
		if r.Runtime != "" {
			extras = append(extras, "rt:"+r.Runtime)
		}
		if r.MaxPoints > 0 {
			extras = append(extras, fmt.Sprintf("≤%gpt", r.MaxPoints))
		}
		fmt.Fprintf(ctx.Stdout, "%-14s %-6s %-32s %s\n", r.Name, orDash(r.Grant), strings.Join(extras, " "), r.Summary)
	}
	return nil
}

func cmdTeam(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	roles, _ := store.LoadRoles(w)
	for _, r := range roles {
		active := store.ActiveInRole(w, r.Name)
		head := "∞"
		if r.WIP > 0 {
			head = fmt.Sprint(r.WIP - active)
		}
		fmt.Fprintf(ctx.Stdout, "%-14s active:%d headroom:%s  %s\n", r.Name, active, head, r.Summary)
	}
	agents, _ := store.ListAgents(w)
	unroled := 0
	for _, a := range agents {
		if a.Role == "" && !a.Retired {
			unroled++
		}
	}
	if unroled > 0 {
		fmt.Fprintf(ctx.Stdout, "(plus %d agents with no role)\n", unroled)
	}
	return nil
}

func cmdTeamRoute(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli team route <path> [--from role]")
	}
	roles, _ := store.LoadRoles(w)
	if len(roles) == 0 {
		return fmt.Errorf("no roles defined; `dacli role add` first")
	}
	tm, err := team.New(roles)
	if err != nil {
		return err
	}
	path := f.pos[0]

	owners := tm.Owners(path)
	if len(owners) == 0 {
		fmt.Fprintf(ctx.Stdout, "no role covers %s — escalate to a human\n", path)
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "owners (most specific first): %s\n", strings.Join(owners, ", "))

	from := f.get("from")
	if from == "" {
		from = id.Role
	}
	if from != "" && from != "root" {
		chain, err := tm.Route(from, path)
		if err != nil {
			// The G8 rule: an owner that exists but is unreachable is a
			// missing edge, not a dead end — the message must say which.
			return fmt.Errorf("%s owns this but is not reachable from %q's escalation chain — add it to escalate_to, or route via a shared ancestor (%w)", owners[0], from, err)
		}
		fmt.Fprintf(ctx.Stdout, "chain from %s: %s\n", from, strings.Join(chain, " → "))
	}
	return nil
}

func cmdAgentRetire(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli agent retire <agent-id>")
	}
	if id.Grant != model.GrantRW {
		return refusedf("retiring an agent rewrites its file, which needs an rw grant")
	}
	if err := store.RetireAgent(w, f.pos[0]); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "retired %s (lineage and attribution kept; WIP slot freed)\n", f.pos[0])
	return nil
}

// cmdDoctor runs anti-pattern detectors over tasks, risks, and the event
// log. Informational: it never fails the build — the point is to make the
// SPM anti-patterns visible while they are still cheap.
func cmdDoctor(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
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
	for _, t := range tasks {
		switch t.Status {
		case model.StatusDone:
			done++
		case model.StatusActive:
			active++
			if model.Priority(t.Priority()).Rank() > 0 {
				lowerActive = append(lowerActive, fmt.Sprintf("%03d-%s(%s)", t.Seq, t.Slug, orDash(t.Priority())))
			}
		case model.StatusOpen:
			if model.Priority(t.Priority()).Rank() == 0 && t.Priority() != "" {
				mustsOpen++
			}
		}
	}

	// Cart Before the Horse — the most common agent planning failure.
	if mustsOpen > 0 && len(lowerActive) > 0 {
		report("cart-before-the-horse", fmt.Sprintf("%d must task(s) sit open while lower-priority work is active: %s",
			mustsOpen, strings.Join(lowerActive, ", ")))
	}

	// Burning Across — tasks started, none finished.
	if active >= 3 && done == 0 {
		report("burning-across", fmt.Sprintf("%d tasks active, 0 done — finish before starting; redirect free agents to help", active))
	}

	// Analysis Paralysis — findings pile up, nothing ships.
	findings, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventFinding}})
	noteFindings := 0
	if ps, _ := store.ListProjects(w); ps != nil {
		for _, p := range ps {
			ns, _ := store.ListNotes(w, p.Slug, model.NoteFinding)
			noteFindings += len(ns)

			// Rank-1 risks with no action plan.
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

	// Unanswered questions block tasks silently.
	if qs, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}, Pending: true}); len(qs) > 0 {
		report("unanswered-questions", fmt.Sprintf("%d question(s) open — the asking tasks are blocked until someone answers", len(qs)))
	}

	// WIP breaches (a spawn refusal exists, but roles can be edited under
	// running agents).
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

// cmdStandup is the daily-scrum roll-up, derived entirely from the log and
// the tasks — no agent ever fills in a status report.
func cmdStandup(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
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

// cmdRetro harvests a completed task or project into a durable note — the
// findings-harvest step, in the went-well / didn't / improve order because
// the order is the technique.
func cmdRetro(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 || (len(f.all("well"))+len(f.all("bad"))+len(f.all("improve"))) == 0 {
		return usagef("usage: dacli retro <task-or-project-ref> --well x [--well ...] --bad y --improve z")
	}
	ref := f.pos[0]
	project := ref
	about := ref
	if t, err := store.FindTask(w, ref); err == nil {
		project = t.Project
		about = t.ID
	} else if _, err := store.LoadProject(w, ref); err != nil {
		return store.ErrNotFound{Ref: ref}
	}

	bullets := func(items []string) string {
		var b strings.Builder
		for _, it := range items {
			b.WriteString("- " + it + "\n")
		}
		return b.String()
	}
	body := "## Went well\n" + bullets(f.all("well")) +
		"\n## Didn't go well\n" + bullets(f.all("bad")) +
		"\n## Improve next time\n" + bullets(f.all("improve"))

	path, err := store.CreateNote(w, id.ID, project, model.NoteRef, "Retro: "+ref, store.NoteOpts{
		About: about,
		Body:  body,
		Scope: f.get("scope"), // --scope workspace makes it a cross-project lesson
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "retro recorded: %s\n", path)
	return nil
}
