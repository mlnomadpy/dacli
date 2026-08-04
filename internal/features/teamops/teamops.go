// Package teamops is the org slice: agent identities and lineage, roles as
// mechanical capability bundles, and escalation routing.
package teamops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "agent spawn", Brief: "Mint a child agent identity and print its token once", Run: cmdAgentSpawn},
	{Path: "agent tree", Brief: "Show agent lineage, roles, current task and write attribution", Run: cmdAgentTree},
	{Path: "agent show", Brief: "Resolve one agent id: role, lineage, runs, tasks, events", Run: cmdAgentShow},
	{Path: "agent retire", Brief: "Mark an agent retired, freeing its WIP slot", Run: cmdAgentRetire},
	{Path: "role add", Brief: "Define a role: skills, scope, shortcuts, escalation", Run: cmdRoleAdd},
	{Path: "role list", Brief: "List roles", Run: cmdRoleList},
	{Path: "role show", Brief: "One role: version, changelog, capabilities", Run: cmdRoleShow},
	{Path: "role bump", Brief: "Increment a role's version (v1→v2) after a change", Run: cmdRoleBump},
	{Path: "team", Brief: "Roster: roles, active agents, WIP headroom", Run: cmdTeam},
	{Path: "team route", Brief: "Who owns this path, and the chain to reach them", Run: cmdTeamRoute},
	{Path: "team assign", Brief: "Which role should take this task: the cheapest model whose capacity covers its Te, for the phase's allowed kind", Run: cmdTeamAssign},
}

func cmdAgentSpawn(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("grant", "role"); err != nil {
		return err
	}
	grant := model.Grant(f.Get("grant"))
	roleName := f.Get("role")

	// A role file supplies defaults and limits. Its grant is a ceiling
	// REQUEST — attenuation against the parent still wins in Spawn.
	var roleSkills, roleShortcuts []string
	if role, ok := store.LoadRole(w, roleName); ok {
		if grant == "" && role.Grant != "" {
			grant = model.Grant(role.Grant)
		}
		roleSkills, roleShortcuts = role.Skills, role.Shortcuts
		if role.WIP > 0 {
			if active := store.ActiveInRole(w, roleName); active >= role.WIP {
				// Burning Across made preventable rather than detectable:
				// the refusal happens BEFORE the thirty-first child exists.
				return clikit.Refusedf("role %s is at its WIP limit (%d/%d) — `dacli agent retire` one, or raise wip in the role file",
					roleName, active, role.WIP)
			}
		}
	}

	childID, token, err := agentid.Spawn(w, id, roleName, grant)
	if err != nil {
		if err == agentid.ErrAttenuation {
			return clikit.Refusedf("%v: your grant is %s", err, id.Grant)
		}
		return err
	}
	if len(roleSkills) > 0 {
		fmt.Fprintf(ctx.Stderr, "role skills to load for the child: %s\n", strings.Join(roleSkills, ", "))
	}
	if len(roleShortcuts) > 0 {
		fmt.Fprintf(ctx.Stderr, "role toolkit: %s\n", strings.Join(roleShortcuts, ", "))
	}
	// The token goes to stdout ALONE so `TOKEN=$(dacli agent spawn ...)`
	// captures exactly it; everything human-facing goes to stderr. It is
	// shown once and never stored — a lost token means a new agent.
	fmt.Fprintln(ctx.Stdout, token)
	fmt.Fprintf(ctx.Stderr, "spawned %s (role: %s, grant: %s)\ntoken shown once above — pass it to the child as %s\n",
		childID, clikit.OrDash(f.Get("role")), clikit.OrDash(string(grant), "ro"), agentid.EnvVar)
	return nil
}

func cmdAgentTree(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(w.AgentsDir())
	if err != nil {
		return err
	}

	type agent struct {
		id, parent, grant, role string
		retired                 bool
	}
	byParent := map[string][]agent{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		d, err := mdstore.ReadFile(w.AgentPath(strings.TrimSuffix(e.Name(), ".md")))
		if err != nil {
			continue
		}
		a := agent{}
		a.id, _ = d.Front.Get("id")
		a.grant, _ = d.Front.Get("grant")
		a.role, _ = d.Front.Get("role")
		if p, ok := d.Front.Get("parent"); ok {
			a.parent = strings.TrimSuffix(strings.TrimPrefix(p, "[["), "]]")
		}
		if r, _ := d.Front.Get("retired"); r == "true" {
			a.retired = true
		}
		byParent[a.parent] = append(byParent[a.parent], a)
	}

	// Write attribution: how many events each agent has actually produced.
	writes := map[string]int{}
	events, _ := eventlog.List(w, eventlog.Query{})
	for _, e := range events {
		writes[e.Actor]++
	}
	// Traceability (dacli 225): the run records already say which task each
	// agent was spawned against and which run produced its work. Joining them
	// here is what turns the tree from a list of names into something an
	// operator can act on without opening a single file.
	runs := runsByAgent(w)

	var render func(a agent, depth int)
	render = func(a agent, depth int) {
		role := a.role
		if role == "" {
			// An agent file with no role: the id itself may still carry one,
			// which is the whole point of a readable id.
			role = agentid.RoleOf(a.id)
		}
		if role != "" {
			role = " · " + role
		}
		fmt.Fprintf(ctx.Stdout, "%s%s (%s%s) — %d events%s\n",
			strings.Repeat("  ", depth), a.id, a.grant, role, writes[a.id], traceSuffix(w, runs[a.id], a.retired))
		kids := byParent[a.id]
		sort.Slice(kids, func(i, j int) bool { return kids[i].id < kids[j].id })
		for _, k := range kids {
			render(k, depth+1)
		}
	}
	for _, root := range byParent[""] {
		render(root, 0)
	}
	return nil
}

// runsByAgent indexes the run records by the agent they spawned, newest run
// first (dacli 225). This is the join that makes an id traceable: a run record
// already carries the task, the runtime, the claimed paths and the run id under
// which the brief, invocation and outcome were filed. No new file format — just
// reading the one that has been written at every spawn since E2.
func runsByAgent(w *workspace.Workspace) map[string][]procmon.Record {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names))) // ULID dir names: newest first
	out := map[string][]procmon.Record{}
	for _, n := range names {
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(n), "proc.txt"))
		if err != nil || rec.Child == "" {
			continue
		}
		if rec.RunID == "" {
			rec.RunID = n
		}
		out[rec.Child] = append(out[rec.Child], rec)
	}
	return out
}

// taskLabel renders a run record's task ref the way humans refer to tasks
// (042-fix-the-thing), falling back to the raw ref when the task has since been
// archived — a dangling ref is still better traceability than nothing.
func taskLabel(w *workspace.Workspace, ref string) string {
	if ref == "" {
		return ""
	}
	if t, err := store.FindTask(w, ref); err == nil {
		return fmt.Sprintf("%03d-%s", t.Seq, t.Slug)
	}
	return ref
}

// traceSuffix appends the newest run's task and run id to a tree line, so the
// tree answers "what is this agent doing, and where is its work recorded".
func traceSuffix(w *workspace.Workspace, recs []procmon.Record, retired bool) string {
	var parts []string
	if len(recs) > 0 {
		if lbl := taskLabel(w, recs[0].Task); lbl != "" {
			parts = append(parts, "task "+lbl)
		}
		parts = append(parts, "run "+recs[0].RunID)
	}
	if retired {
		parts = append(parts, "retired")
	}
	if len(parts) == 0 {
		return ""
	}
	return " · " + strings.Join(parts, " · ")
}

// cmdAgentShow resolves ONE id all the way: role, lineage, grant, the runs it
// was spawned for and the work it recorded (dacli 225). This is the command an
// operator reaches for after reading an unfamiliar id in `git log` or a task's
// Log — the alternative it replaces is opening .dacli/agents/<id>.md by hand and
// then grepping the runs dir.
func cmdAgentShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli agent show <agent-id>")
	}
	want := f.Pos[0]

	d, derr := mdstore.ReadFile(w.AgentPath(want))
	if derr != nil {
		// An id with no file is not necessarily a typo: an agent can be named in
		// a commit trailer of a checkout whose .dacli/agents/ predates it. Say
		// what the id itself still reveals rather than only "not found".
		if role := agentid.RoleOf(want); role != "" {
			return store.ErrNotFound{Ref: "agent " + want + " (id reads as role " + role + "; the agent file is not in this checkout)"}
		}
		return store.ErrNotFound{Ref: "agent " + want}
	}
	get := func(k string) string { v, _ := d.Front.Get(k); return v }

	role := get("role")
	if role == "" {
		role = agentid.RoleOf(want)
	}
	fmt.Fprintf(ctx.Stdout, "%s\n", want)
	fmt.Fprintf(ctx.Stdout, "%-9s %s\n", "role:", clikit.OrDash(role))
	fmt.Fprintf(ctx.Stdout, "%-9s %s\n", "grant:", clikit.OrDash(get("grant")))
	fmt.Fprintf(ctx.Stdout, "%-9s %s\n", "created:", clikit.OrDash(get("created")))
	if get("retired") == "true" {
		fmt.Fprintf(ctx.Stdout, "%-9s yes (lineage and attribution kept)\n", "retired:")
	}

	// Lineage, walked to the root: "who delegated this" is the question an
	// unfamiliar id raises first, and one hop rarely answers it.
	chain := []string{want}
	seen := map[string]bool{want: true}
	for cur := want; ; {
		cd, err := mdstore.ReadFile(w.AgentPath(cur))
		if err != nil {
			break
		}
		p, ok := cd.Front.Get("parent")
		if !ok {
			break
		}
		p = strings.TrimSuffix(strings.TrimPrefix(p, "[["), "]]")
		// A cycle would be corruption, but a corrupt file must not hang a
		// read-only command.
		if p == "" || seen[p] {
			break
		}
		seen[p] = true
		chain = append([]string{p}, chain...)
		cur = p
	}
	fmt.Fprintf(ctx.Stdout, "%-9s %s\n", "lineage:", strings.Join(chain, " → "))

	recs := runsByAgent(w)[want]
	if len(recs) == 0 {
		fmt.Fprintf(ctx.Stdout, "%-9s none recorded (spawned outside `dacli spawn`, or pre-E2)\n", "runs:")
	} else {
		fmt.Fprintf(ctx.Stdout, "runs (newest first):\n")
		for _, r := range recs {
			line := "  " + r.RunID
			if lbl := taskLabel(w, r.Task); lbl != "" {
				line += "  task " + lbl
			}
			if r.Runtime != "" {
				line += "  runtime " + r.Runtime
			}
			if !r.Started.IsZero() {
				line += "  started " + r.Started.UTC().Format(time.RFC3339)
			}
			if len(r.Claims) > 0 {
				line += "  claims " + strings.Join(r.Claims, ",")
			}
			fmt.Fprintln(ctx.Stdout, line)
		}
	}

	events, _ := eventlog.List(w, eventlog.Query{Actor: want})
	fmt.Fprintf(ctx.Stdout, "events:   %d\n", len(events))
	for i, e := range events {
		if i == 5 {
			fmt.Fprintf(ctx.Stdout, "  … %d more\n", len(events)-i)
			break
		}
		fmt.Fprintf(ctx.Stdout, "  %s %s %s\n", e.ID, e.Kind, clikit.OrDash(clikit.FirstLine(e.Body)))
	}
	return nil
}

func cmdAgentRetire(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli agent retire <agent-id>")
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("retiring an agent rewrites its file, which needs an rw grant")
	}
	if err := store.RetireAgent(w, f.Pos[0]); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "retired %s (lineage and attribution kept; WIP slot freed)\n", f.Pos[0])
	return nil
}

func cmdRoleAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("summary", "kind", "skill", "scope", "out-of-scope", "shortcut", "escalate-to", "grant", "wip", "runtime", "model", "max-points"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli role add <name> [--summary s] [--kind researcher|planner|designer|implementer|reviewer] [--skill s]... [--scope glob]... [--shortcut n]... [--escalate-to role]... [--grant ro|rw] [--wip N] [--runtime rt] [--model m] [--max-points N]")
	}
	r := team.Role{
		Name:       f.Pos[0],
		Summary:    f.Get("summary"),
		Skills:     f.All("skill"),
		Scope:      f.All("scope"),
		OutOfScope: f.All("out-of-scope"),
		Shortcuts:  f.All("shortcut"),
		EscalateTo: f.All("escalate-to"),
		Grant:      f.Get("grant"),
		Kind:       f.Get("kind"),
		Runtime:    f.Get("runtime"),
		Model:      f.Get("model"),
	}
	if r.WIP, err = f.Int("wip", 0); err != nil {
		return err
	}
	fmt.Sscanf(f.Get("max-points"), "%g", &r.MaxPoints)

	// A role must change what an agent can do, not just what it calls
	// itself. A name-only role is cosplay; warn, don't refuse — the fields
	// can be added later, but the warning should sting now.
	if len(r.Skills)+len(r.Scope)+len(r.Shortcuts)+len(r.EscalateTo) == 0 && r.Grant == "" && r.WIP == 0 && r.Model == "" && r.Runtime == "" && r.MaxPoints == 0 && r.Kind == "" {
		fmt.Fprintln(ctx.Stderr, "warning: this role changes nothing mechanical (no skills, scope, shortcuts, escalation, grant, or wip) — it is a costume, not a role")
	}
	if err := store.CreateRole(w, id.ID, r); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "role %s defined\n", r.Name)
	return nil
}

func cmdRoleList(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
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
		if r.Kind != "" {
			extras = append(extras, "kind:"+r.Kind)
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
		fmt.Fprintf(ctx.Stdout, "%-14s %-6s %-32s %s\n", r.Name, clikit.OrDash(r.Grant), strings.Join(extras, " "), r.Summary)
	}
	return nil
}

func cmdRoleShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli role show <name>")
	}
	name := f.Pos[0]
	r, ok := store.LoadRole(w, name)
	if !ok {
		return store.ErrNotFound{Ref: "role " + name}
	}
	path := w.RolePath(name)
	version := store.FileVersion(path)

	fmt.Fprintf(ctx.Stdout, "%s — %s\nversion: %s · grant: %s\n", r.Name, clikit.OrDash(r.Summary), version, clikit.OrDash(r.Grant))
	field := func(label string, vals []string) {
		if len(vals) > 0 {
			fmt.Fprintf(ctx.Stdout, "%-12s %s\n", label+":", strings.Join(vals, ", "))
		}
	}
	field("skills", r.Skills)
	field("scope", r.Scope)
	field("out-of-scope", r.OutOfScope)
	field("shortcuts", r.Shortcuts)
	field("escalate-to", r.EscalateTo)
	if r.Kind != "" {
		fmt.Fprintf(ctx.Stdout, "%-12s %s\n", "kind:", r.Kind)
	}
	if r.WIP > 0 {
		fmt.Fprintf(ctx.Stdout, "%-12s %d\n", "wip:", r.WIP)
	}

	if stale, since := store.VersionIsStale(path, version); stale {
		if since > 0 {
			fmt.Fprintf(ctx.Stdout, "⚠ changed in %d commit(s) since %s was set — bump with `dacli role bump %s`\n", since, version, name)
		} else {
			fmt.Fprintf(ctx.Stdout, "⚠ uncommitted edits — bump with `dacli role bump %s` before committing\n", name)
		}
	}
	changes, seen := store.FileChangelog(path, 10)
	fmt.Fprintf(ctx.Stdout, "\nchangelog:\n%s\n", store.FormatChangelog(changes, seen))
	return nil
}

func cmdRoleBump(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("bumping a role version rewrites its file, which needs an rw grant")
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli role bump <name>")
	}
	name := f.Pos[0]
	if _, ok := store.LoadRole(w, name); !ok {
		return store.ErrNotFound{Ref: "role " + name}
	}
	old, next, err := store.BumpFileVersion(w.RolePath(name))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "role %s: %s → %s — commit it with `dacli commit`\n", name, old, next)
	return nil
}

func cmdTeam(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
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

// cmdTeamAssign answers "who should take this task, and on which model" from
// the task's own size rather than an operator's habit.
//
// dacli already had every piece and used none of them together: roles carry a
// Model tier and a MaxPoints capacity, and the seniority gate REFUSES a role
// whose capacity is below the task's Te. But nothing ever CHOSE a role, and no
// role in a shipped roster declared a capacity — so the gate was inert and
// model tiering was a per-spawn decision made by hand. With capacities
// declared, the choice is mechanical: the cheapest model that can hold the work
// takes it, and the expensive models stay free for the work that needs them
// (dacli 230, 231).
func cmdTeamAssign(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("kind"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli team assign <task-ref> [--kind implementer|reviewer|researcher|planner|designer]")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	tp, sized := t.Estimate()
	if !sized {
		return clikit.Refusedf("task %03d-%s has no estimate — capacity routing needs one; `dacli task estimate %03d --estimate o,m,p`", t.Seq, t.Slug, t.Seq)
	}
	te := tp.Expected()

	// The phase decides which KIND may act; the estimate decides which of those
	// roles is big enough. An explicit --kind overrides the phase, for asking
	// "who would review this" while the project is still implementing.
	kind := f.Get("kind")
	if kind == "" {
		kind = "implementer"
		if ph, err := gates.PhaseFor(w, t.Project); err == nil && ph.Gated && len(ph.Allows) > 0 {
			kind = ph.Allows[0]
		}
	}

	roles, _ := store.LoadRoles(w)
	pick, ok := team.CheapestCapable(roles, kind, te, t.PathHints())
	if !ok {
		return clikit.Refusedf("no %s role can hold Te %.1f — every capped role is too small and none is uncapped; decompose %03d-%s or add a heavier role",
			kind, te, t.Seq, t.Slug)
	}

	cap := "uncapped"
	if pick.MaxPoints > 0 {
		cap = fmt.Sprintf("cap %g", pick.MaxPoints)
	}
	fmt.Fprintf(ctx.Stdout, "%03d-%s (Te %.1f) → %s  [%s · %s]\n", t.Seq, t.Slug, te, pick.Name,
		clikit.OrDash(pick.Model), cap)
	fmt.Fprintf(ctx.Stdout, "  cheapest %s whose capacity covers it\n", kind)
	fmt.Fprintf(ctx.Stdout, "  dacli spawn --task %03d --role %s\n", t.Seq, pick.Name)
	return nil
}

func cmdTeamRoute(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("from"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli team route <path> [--from role]")
	}
	roles, _ := store.LoadRoles(w)
	if len(roles) == 0 {
		return fmt.Errorf("no roles defined; `dacli role add` first")
	}
	tm, err := team.New(roles)
	if err != nil {
		return err
	}
	path := f.Pos[0]

	owners := tm.Owners(path)
	if len(owners) == 0 {
		fmt.Fprintf(ctx.Stdout, "no role covers %s — escalate to a human\n", path)
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "owners (most specific first): %s\n", strings.Join(owners, ", "))

	from := f.Get("from")
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
