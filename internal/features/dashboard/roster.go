package dashboard

import (
	"sort"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// The role roster (dacli 226).
//
// A role is the only thing in dacli that mechanically changes what an agent can
// do — which skills load, which paths are in scope, which runtime and model it
// costs, how many of it may run at once (see internal/team's package doc). Until
// now that whole configuration was invisible from the dashboard: an operator who
// wanted to know why a spawn was refused, or which role owns a subtree, had to
// open .dacli/roles/*.md by hand. This surface is the roster those files
// describe, read the same way `dacli role list` reads them.

// rolesResponse is GET /api/roles: the workspace roster, sorted by name so the
// list is stable across polls (LoadRoles returns directory order, which is
// stable in practice but not promised). The same []roleView is embedded in
// /api/state, so the standalone surface and the combined snapshot can never
// disagree.
type rolesResponse struct {
	Generated string     `json:"generated"`
	Roles     []roleView `json:"roles"`
}

// roleView is one role on the roster: its identity, the four mechanical levers
// (scope, grant, runtime/model, caps), and how much of its WIP budget is spent
// right now.
//
// Every slice field is emitted as [] rather than null (see strs) — an absent
// boundary and an empty one mean the same thing to a reader, and a client that
// has to null-check every list is a client that will forget once.
type roleView struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	// Kind is the lifecycle function phase gating acts on (researcher, planner,
	// designer, implementer, reviewer). Empty means the role opts out of gating.
	Kind string `json:"kind"`
	// Grant is the default capability a spawn into this role receives.
	Grant string `json:"grant"`
	// Runtime and Model are where cost policy lives: a reviewer can demand the
	// expensive model while a junior runs on the cheap one.
	Runtime string `json:"runtime"`
	Model   string `json:"model"`
	// WIP caps concurrent agents in this role; 0 means uncapped.
	WIP int `json:"wip"`
	// MaxPoints caps the expected size (Te) of tasks the role may take; 0 means
	// uncapped.
	MaxPoints  float64  `json:"max_points"`
	Scope      []string `json:"scope"`
	OutOfScope []string `json:"out_of_scope"`
	Skills     []string `json:"skills"`
	Shortcuts  []string `json:"shortcuts"`
	EscalateTo []string `json:"escalate_to"`
	// ActiveAgents is how many non-retired agents currently hold this role — the
	// WIP numerator, counted exactly as store.ActiveInRole counts it.
	ActiveAgents int `json:"active_agents"`
	// WIPExceeded is the server's verdict that another spawn into this role would
	// break its cap, computed by the same team.WIPExceeded the spawn path uses so
	// the dashboard and the refusal can never disagree. False for an uncapped role.
	WIPExceeded bool `json:"wip_exceeded"`
	// HasPrompt reports whether the role file carries standing instructions below
	// its frontmatter — the difference between a role that is DESCRIBED and one
	// that is DEFINED (see store.roleBody). The body itself is deliberately not
	// served: it is prose measured in kilobytes, and a roster is a list, not a
	// reader. `dacli role show <name>` prints it.
	HasPrompt bool `json:"has_prompt"`
}

// buildRoles reads every role file and joins it to the live agent census.
//
// The census is taken with ONE store.ListAgents pass rather than a
// store.ActiveInRole call per role: ActiveInRole re-reads the entire agents
// directory each time it is asked, so per-role calls would cost roles×agents
// file reads on a surface that is polled every two seconds (this workspace
// alone: 17 roles × 225 agents). The counting rule is ActiveInRole's verbatim —
// same role, not retired.
func buildRoles(w *workspace.Workspace) (rolesResponse, error) {
	resp := rolesResponse{Generated: nowStamp(), Roles: []roleView{}}
	roles, err := store.LoadRoles(w)
	if err != nil {
		return resp, err
	}
	active := activeByRole(w)
	// team.New rejects a duplicate/unnamed role, which is a broken roster rather
	// than a dashboard failure — fall back to counting without the WIP verdict so
	// a misconfigured workspace still renders its roles instead of 500ing.
	t, _ := team.New(roles)
	for _, r := range roles {
		rv := roleView{
			Name: r.Name, Summary: r.Summary, Kind: r.Kind,
			Grant: r.Grant, Runtime: r.Runtime, Model: r.Model,
			WIP: r.WIP, MaxPoints: r.MaxPoints,
			Scope:      strs(r.Scope),
			OutOfScope: strs(r.OutOfScope),
			Skills:     strs(r.Skills),
			Shortcuts:  strs(r.Shortcuts),
			EscalateTo: strs(r.EscalateTo),

			ActiveAgents: active[r.Name],
			HasPrompt:    r.Prompt != "",
		}
		if t != nil {
			rv.WIPExceeded = t.WIPExceeded(r.Name, rv.ActiveAgents)
		}
		resp.Roles = append(resp.Roles, rv)
	}
	sort.Slice(resp.Roles, func(i, j int) bool { return resp.Roles[i].Name < resp.Roles[j].Name })
	return resp, nil
}

// activeByRole counts non-retired agents per role in a single directory pass.
// An agent file's role is the only field read here; nothing else about the agent
// (least of all its token_hash) reaches this surface, because store.AgentInfo
// does not carry it.
func activeByRole(w *workspace.Workspace) map[string]int {
	out := map[string]int{}
	agents, err := store.ListAgents(w)
	if err != nil {
		// No agents dir yet is an empty census, not a dashboard failure: a fresh
		// workspace has roles before it has agents.
		return out
	}
	for _, a := range agents {
		if !a.Retired {
			out[a.Role]++
		}
	}
	return out
}

// strs normalizes a possibly-nil string slice to a non-nil one so it marshals as
// [] instead of null. The JSON shapes here are a contract; a field that is
// sometimes a list and sometimes null is two contracts.
func strs(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
