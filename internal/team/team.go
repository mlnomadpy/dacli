// Package team implements roles, scope boundaries, and escalation routing
// for a tree of agents.
//
// The design rule that keeps roles from becoming cosplay: A ROLE MUST CHANGE
// WHAT AN AGENT CAN DO, NOT JUST WHAT IT CALLS ITSELF. Prepending "You are a
// senior frontend engineer" to a prompt is theater — it costs tokens and
// changes nothing mechanical. A role here determines which skills load, which
// shortcuts are reachable, which paths are in scope, and what must be
// escalated rather than attempted. If a proposed role changes none of those,
// it should not exist.
package team

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

// Role is a named function on the team.
type Role struct {
	Name    string `yaml:"name"`
	Summary string `yaml:"summary"`

	// Skills are loaded into the agent's context at spawn. These name skills
	// in the workspace library (.dacli/skills/), which are compiled to the
	// target runtime's delivery mechanism — native skill dir, context file,
	// or brief-inline — at spawn time. See docs/SKILLS.md.
	Skills []string `yaml:"skills,omitempty"`

	// Scope and OutOfScope are path globs supporting ** for "any number of
	// segments". OutOfScope wins on conflict, because a deny that can be
	// overridden by a broader allow is not a boundary.
	Scope      []string `yaml:"scope,omitempty"`
	OutOfScope []string `yaml:"out_of_scope,omitempty"`

	// Shortcuts this role may run.
	Shortcuts []string `yaml:"shortcuts,omitempty"`

	// Grant is the default capability when spawning into this role.
	Grant string `yaml:"grant,omitempty"`

	// EscalateTo is the ordered list of roles to ask when work falls outside
	// Scope. The literal "human" terminates the chain.
	EscalateTo []string `yaml:"escalate_to,omitempty"`

	// WIP caps concurrent agents in this role. Borrowed from Kanban, and it
	// is the only thing standing between an enthusiastic parent agent and
	// thirty children contending over four files.
	WIP int `yaml:"wip,omitempty"`

	// Kind is the role's function in the project lifecycle: researcher,
	// planner, designer, implementer, reviewer. It is what phase gating acts
	// on — an implementer cannot be spawned during discovery. A role with no
	// kind opts out of phase gating (works in any phase).
	Kind string `yaml:"kind,omitempty"`

	// Runtime and Model route this role onto a coding-agent CLI and a model
	// tier. This is where cost policy lives: a reviewer role can demand the
	// expensive model while a junior role runs on the cheap one — and the
	// difference is mechanical, not aspirational.
	Runtime string `yaml:"runtime,omitempty"`
	Model   string `yaml:"model,omitempty"`

	// MaxPoints caps the expected size (Te) of tasks this role may take.
	// A junior role with MaxPoints 3 mechanically cannot be spawned onto
	// the hard migration — the refusal names a heavier role instead. Zero
	// means uncapped.
	MaxPoints float64 `yaml:"max_points,omitempty"`
}

// Human is the terminal escalation target: no agent in the tree can answer,
// so it leaves the tree entirely.
const Human = "human"

// InScope reports whether a path falls inside this role's boundary.
//
// Deny beats allow. An empty Scope means "no declared boundary", which is
// permissive by design — most small projects want one role and no fences, and
// forcing everyone to enumerate their scope up front would just produce
// wrong globs written to satisfy the linter.
func (r Role) InScope(p string) bool {
	p = path.Clean(strings.TrimPrefix(p, "./"))
	for _, g := range r.OutOfScope {
		if matchGlob(g, p) {
			return false
		}
	}
	if len(r.Scope) == 0 {
		return true
	}
	for _, g := range r.Scope {
		if matchGlob(g, p) {
			return true
		}
	}
	return false
}

// CanRun reports whether this role may run the named shortcut.
func (r Role) CanRun(name string) bool {
	if len(r.Shortcuts) == 0 {
		return true
	}
	for _, s := range r.Shortcuts {
		if s == name {
			return true
		}
	}
	return false
}

// matchGlob matches a slash-separated path against a glob supporting ** for
// zero or more segments, and * within a single segment.
func matchGlob(pattern, p string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(p, "/"))
}

func matchSegments(pat, seg []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			// Trailing ** matches everything remaining.
			if len(pat) == 1 {
				return true
			}
			for i := 0; i <= len(seg); i++ {
				if matchSegments(pat[1:], seg[i:]) {
					return true
				}
			}
			return false
		}
		if len(seg) == 0 {
			return false
		}
		if ok, err := path.Match(pat[0], seg[0]); err != nil || !ok {
			return false
		}
		pat, seg = pat[1:], seg[1:]
	}
	return len(seg) == 0
}

// Team is the roster.
type Team struct {
	Roles map[string]Role
}

// New builds a Team from a role list.
func New(roles []Role) (*Team, error) {
	t := &Team{Roles: make(map[string]Role, len(roles))}
	for _, r := range roles {
		if r.Name == "" {
			return nil, errors.New("role with no name")
		}
		if _, dup := t.Roles[r.Name]; dup {
			return nil, fmt.Errorf("duplicate role %q", r.Name)
		}
		t.Roles[r.Name] = r
	}
	return t, nil
}

// ErrNoOwner means nothing in the tree covers the request, so it escalates
// out to a human. This is a normal outcome, not a failure: an agent tree that
// can never say "I don't know who handles this" will instead have somebody
// guess, and the guess ships.
var ErrNoOwner = errors.New("no role covers this path; escalate to a human")

// Route returns the escalation chain for work at p, starting from role from.
//
// It follows declared EscalateTo edges breadth-first, returning the first
// role whose scope covers the path. Cycles are handled: a mutual
// escalate_to pair is a configuration mistake, not an infinite loop.
func (t *Team) Route(from string, p string) ([]string, error) {
	start, ok := t.Roles[from]
	if !ok {
		return nil, fmt.Errorf("unknown role %q", from)
	}
	if start.InScope(p) {
		return []string{from}, nil
	}

	seen := map[string]bool{from: true}
	type hop struct {
		name string
		path []string
	}
	queue := []hop{{from, []string{from}}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		role := t.Roles[cur.name]

		for _, next := range role.EscalateTo {
			if next == Human {
				continue
			}
			if seen[next] {
				continue
			}
			seen[next] = true

			cand, ok := t.Roles[next]
			if !ok {
				return nil, fmt.Errorf("role %q escalates to unknown role %q", cur.name, next)
			}
			chain := append(append([]string(nil), cur.path...), next)
			if cand.InScope(p) {
				return chain, nil
			}
			queue = append(queue, hop{next, chain})
		}
	}
	return nil, ErrNoOwner
}

// Owners returns every role whose scope covers p, ordered by specificity:
// the role with the narrowest declared scope wins, since a catch-all role
// should never outrank a specialist.
func (t *Team) Owners(p string) []string {
	var out []string
	for name, r := range t.Roles {
		if r.InScope(p) {
			out = append(out, name)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		li, lj := len(t.Roles[out[i]].Scope), len(t.Roles[out[j]].Scope)
		if li == 0 {
			li = 1 << 30 // no declared scope = maximally general
		}
		if lj == 0 {
			lj = 1 << 30
		}
		if li != lj {
			return li < lj
		}
		return out[i] < out[j]
	})
	return out
}

// WIPExceeded reports whether spawning another agent in this role would break
// its work-in-progress limit.
//
// This is the Burning Across anti-pattern made preventable rather than merely
// detectable: many tasks started, none finished. An agent asked to parallelize
// will spawn as many children as it has tasks, and contention does the rest.
func (t *Team) WIPExceeded(role string, active int) bool {
	r, ok := t.Roles[role]
	if !ok || r.WIP <= 0 {
		return false
	}
	return active >= r.WIP
}
