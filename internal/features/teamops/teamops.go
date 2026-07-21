// Package teamops is the org slice: agent identities and lineage, roles as
// mechanical capability bundles, and escalation routing.
package teamops

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

var Commands = []clikit.Command{
	{Path: "agent spawn", Brief: "Mint a child agent identity and print its token once", Run: cmdAgentSpawn},
	{Path: "agent tree", Brief: "Show agent lineage and write attribution", Run: cmdAgentTree},
	{Path: "agent retire", Brief: "Mark an agent retired, freeing its WIP slot", Run: cmdAgentRetire},
	{Path: "role add", Brief: "Define a role: skills, scope, shortcuts, escalation", Run: cmdRoleAdd},
	{Path: "role list", Brief: "List roles", Run: cmdRoleList},
	{Path: "team", Brief: "Roster: roles, active agents, WIP headroom", Run: cmdTeam},
	{Path: "team route", Brief: "Who owns this path, and the chain to reach them", Run: cmdTeamRoute},
}

func cmdAgentSpawn(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
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
		byParent[a.parent] = append(byParent[a.parent], a)
	}

	// Write attribution: how many events each agent has actually produced.
	writes := map[string]int{}
	events, _ := eventlog.List(w, eventlog.Query{})
	for _, e := range events {
		writes[e.Actor]++
	}

	var render func(a agent, depth int)
	render = func(a agent, depth int) {
		role := a.role
		if role != "" {
			role = " · " + role
		}
		fmt.Fprintf(ctx.Stdout, "%s%s (%s%s) — %d events\n", strings.Repeat("  ", depth), a.id, a.grant, role, writes[a.id])
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
	fmt.Sscanf(f.Get("wip"), "%d", &r.WIP)
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

func cmdTeamRoute(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
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
