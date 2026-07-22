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
	// A role is versioned from birth so `role show` always has something
	// legible to print and every later edit has a baseline to bump past.
	d.Front.Set("version", DefaultVersion)
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
	if r.Kind != "" {
		// role_kind, NOT kind: `kind` is the object-type frontmatter ("role")
		// every file carries. Reusing it made every role read back kind="role"
		// and get gated as an unknown kind. Found by the phase test.
		d.Front.Set("role_kind", r.Kind)
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

// parseRole builds the pure engine's type from a parsed role doc. fallbackName
// is used when the file omits an explicit name (defaults to the filename).
func parseRole(d *mdstore.Doc, fallbackName string) team.Role {
	r := team.Role{}
	r.Name, _ = d.Front.Get("name")
	if r.Name == "" {
		r.Name = fallbackName
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
	r.Kind, _ = d.Front.Get("role_kind")
	r.Runtime, _ = d.Front.Get("runtime")
	r.Model, _ = d.Front.Get("model")
	if mp, ok := d.Front.Get("max_points"); ok {
		fmt.Sscanf(mp, "%g", &r.MaxPoints)
	}
	return r
}

// LoadRoles parses every role file into the pure engine's type.
func LoadRoles(w *workspace.Workspace) ([]team.Role, error) {
	entries, err := os.ReadDir(w.RolesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no roles dir yet is not an error
		}
		return nil, err // a real I/O/permission failure must not read as "empty"
	}
	var out []team.Role
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		d, err := mdstore.ReadFile(w.RolePath(name))
		if err != nil {
			continue
		}
		out = append(out, parseRole(d, name))
	}
	return out, nil
}

// LoadRole reads one role by name from its exact file, rather than scanning the
// whole directory through LoadRoles.
func LoadRole(w *workspace.Workspace, name string) (team.Role, bool) {
	d, err := mdstore.ReadFile(w.RolePath(name))
	if err != nil {
		return team.Role{}, false
	}
	return parseRole(d, name), true
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
