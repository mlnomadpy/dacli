package store

import (
	"fmt"
	"os"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// CreateRole writes .dacli/roles/<name>.md. A role must change what an agent
// can do — skills, scope, shortcuts, escalation — not just what it calls
// itself; a role that sets none of these is cosplay and gets flagged.
func CreateRole(w *workspace.Workspace, actor string, r team.Role) error {
	if r.Name == "" {
		return fmt.Errorf("role needs a name")
	}
	path := w.RolePath(r.Name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("role %q already exists", r.Name)
	}

	d := &mdstore.Doc{}
	d.Front.Set("id", "role-"+r.Name)
	d.Front.Set("kind", string(model.KindRole))
	d.Front.Set("created", now())
	d.Front.Set("created_by", actor)
	d.Front.Set("name", r.Name)
	if r.Summary != "" {
		d.Front.Set("summary", r.Summary)
	}
	setList := func(k string, v []string) {
		if len(v) > 0 {
			d.Front.Set(k, "["+strings.Join(v, ", ")+"]")
		}
	}
	setList("skills", r.Skills)
	setList("scope", r.Scope)
	setList("out_of_scope", r.OutOfScope)
	setList("shortcuts", r.Shortcuts)
	setList("escalate_to", r.EscalateTo)
	if r.Grant != "" {
		d.Front.Set("grant", r.Grant)
	}
	if r.WIP > 0 {
		d.Front.Set("wip", fmt.Sprint(r.WIP))
	}
	if r.Runtime != "" {
		d.Front.Set("runtime", r.Runtime)
	}
	if r.Model != "" {
		d.Front.Set("model", r.Model)
	}
	if r.MaxPoints > 0 {
		d.Front.Set("max_points", fmt.Sprint(r.MaxPoints))
	}
	d.Sections = []mdstore.Section{{Level: 1, Title: r.Name, Content: r.Summary + "\n"}}
	return mdstore.WriteFile(path, d)
}

// LoadRoles parses every role file into the pure engine's type.
func LoadRoles(w *workspace.Workspace) ([]team.Role, error) {
	entries, err := os.ReadDir(w.RolesDir())
	if err != nil {
		return nil, nil
	}
	var out []team.Role
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		d, err := mdstore.ReadFile(w.RolePath(strings.TrimSuffix(e.Name(), ".md")))
		if err != nil {
			continue
		}
		r := team.Role{}
		r.Name, _ = d.Front.Get("name")
		if r.Name == "" {
			r.Name = strings.TrimSuffix(e.Name(), ".md")
		}
		r.Summary, _ = d.Front.Get("summary")
		r.Skills = d.Front.GetList("skills")
		r.Scope = d.Front.GetList("scope")
		r.OutOfScope = d.Front.GetList("out_of_scope")
		r.Shortcuts = d.Front.GetList("shortcuts")
		r.EscalateTo = d.Front.GetList("escalate_to")
		r.Grant, _ = d.Front.Get("grant")
		if wip, ok := d.Front.Get("wip"); ok {
			fmt.Sscanf(wip, "%d", &r.WIP)
		}
		r.Runtime, _ = d.Front.Get("runtime")
		r.Model, _ = d.Front.Get("model")
		if mp, ok := d.Front.Get("max_points"); ok {
			fmt.Sscanf(mp, "%g", &r.MaxPoints)
		}
		out = append(out, r)
	}
	return out, nil
}

// LoadRole finds one role by name.
func LoadRole(w *workspace.Workspace, name string) (team.Role, bool) {
	roles, _ := LoadRoles(w)
	for _, r := range roles {
		if r.Name == name {
			return r, true
		}
	}
	return team.Role{}, false
}

// AgentInfo is the file-level view of an agent, for rosters and standups.
type AgentInfo struct {
	ID      string
	Role    string
	Grant   string
	Parent  string
	Retired bool
}

// ListAgents reads every agent file.
func ListAgents(w *workspace.Workspace) ([]AgentInfo, error) {
	entries, err := os.ReadDir(w.AgentsDir())
	if err != nil {
		return nil, err
	}
	var out []AgentInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		d, err := mdstore.ReadFile(w.AgentPath(strings.TrimSuffix(e.Name(), ".md")))
		if err != nil {
			continue
		}
		a := AgentInfo{}
		a.ID, _ = d.Front.Get("id")
		a.Role, _ = d.Front.Get("role")
		a.Grant, _ = d.Front.Get("grant")
		if p, ok := d.Front.Get("parent"); ok {
			a.Parent = strings.TrimSuffix(strings.TrimPrefix(p, "[["), "]]")
		}
		if r, _ := d.Front.Get("retired"); r == "true" {
			a.Retired = true
		}
		out = append(out, a)
	}
	return out, nil
}

// ActiveInRole counts non-retired agents holding a role — the WIP
// denominator. Agents have no liveness, so "active" means "not retired";
// `agent retire` frees the slot.
func ActiveInRole(w *workspace.Workspace, role string) int {
	agents, _ := ListAgents(w)
	n := 0
	for _, a := range agents {
		if a.Role == role && !a.Retired {
			n++
		}
	}
	return n
}

// RetireAgent marks an agent retired, freeing its WIP slot. The file stays —
// lineage and attribution outlive the agent.
func RetireAgent(w *workspace.Workspace, id string) error {
	d, err := mdstore.ReadFile(w.AgentPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound{Ref: "agent/" + id}
		}
		return err
	}
	d.Front.Set("retired", "true")
	return mdstore.WriteFile(w.AgentPath(id), d)
}
