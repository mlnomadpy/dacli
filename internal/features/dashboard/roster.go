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
	// ActiveAgents is the live, liveness-probed WIP occupancy for this role.
	// Historical identities, including never-started ones, do not consume it.
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
// The census is the same single-pass, liveness-probed store predicate used by
// `dacli team` and the spawn WIP gate. Keeping it below feature slices prevents
// the dashboard API from drifting back to identity-retirement counts (#697).
func buildRoles(w *workspace.Workspace) (rolesResponse, error) {
	resp := rolesResponse{Generated: nowStamp(), Roles: []roleView{}}
	roles, err := store.LoadRoles(w)
	if err != nil {
		return resp, err
	}
	active, err := store.LiveOccupancyByRole(w)
	if err != nil {
		return resp, err
	}
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

// strs normalizes a possibly-nil string slice to a non-nil one so it marshals as
// [] instead of null. The JSON shapes here are a contract; a field that is
// sometimes a list and sometimes null is two contracts.
func strs(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
