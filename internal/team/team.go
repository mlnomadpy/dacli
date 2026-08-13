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
	Runtime string       `yaml:"runtime,omitempty"`
	Model   string       `yaml:"model,omitempty"`
	Profile ModelProfile `yaml:"model_profile,omitempty"`

	// FallbackTo is an explicit ordered chain of role names. A missing chain
	// means stop; dacli never infers a vendor or model substitute.
	FallbackTo []string `yaml:"fallback_to,omitempty"`

	// MaxPoints caps the expected size (Te) of tasks this role may take.
	// A junior role with MaxPoints 3 mechanically cannot be spawned onto
	// the hard migration — the refusal names a heavier role instead. Zero
	// means uncapped.
	MaxPoints float64 `yaml:"max_points,omitempty"`

	// Prompt is the role's standing instructions: HOW an agent in this role
	// works — its method, what it looks for, what it refuses, what "done"
	// means to it. It is the markdown body of the role file, below the
	// frontmatter.
	//
	// Until dacli 202 this body was parsed and discarded, so a role was pure
	// metadata: routing (Scope), permission (Grant), and cost (Runtime/Model).
	// Every role therefore behaved identically — a frontend-engineer and a
	// go-auditor received the same generic prompt templates and differed only
	// in which paths they could touch. Specialization was a filing convention,
	// not expertise. The brief now carries this verbatim, which is what makes
	// a roster of roles a team rather than a directory.
	Prompt string `yaml:"-"`
}

// ModelProfile describes routing facts without coupling them to a provider or
// transport. CostTier is an ordering rank (1 cheapest, 98 dearest); zero and
// values outside that range are deliberately unpriced. MaxTaskPoints zero is
// uncapped and ContextLimit zero means undeclared.
type ModelProfile struct {
	ID             string   `yaml:"model_id,omitempty"`
	CostTier       int      `yaml:"cost_tier,omitempty"`
	MaxTaskPoints  float64  `yaml:"max_task_points,omitempty"`
	ContextLimit   int      `yaml:"context_limit,omitempty"`
	CapabilityTags []string `yaml:"capability_tags,omitempty"`
}

func (r Role) ModelID() string {
	if r.Profile.ID != "" {
		return r.Profile.ID
	}
	return r.Model
}

func (r Role) TaskCapacity() float64 {
	if r.Profile.MaxTaskPoints > 0 {
		return r.Profile.MaxTaskPoints
	}
	return r.MaxPoints
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

// ScopeOverlap counts how many of the given paths fall inside this role's
// DECLARED boundary. A role with no Scope declared scores zero here even
// though InScope would admit every path: an undeclared boundary is generic,
// not a domain match, so it must not out-rank a role that actually named the
// task's files. OutOfScope still excludes a path (via InScope), so a role
// fenced OUT of the work scores it as no overlap. This is the specialization
// signal that breaks a cost+capacity routing tie before name (dacli 238).
func (r Role) ScopeOverlap(paths []string) int {
	if len(r.Scope) == 0 {
		return 0
	}
	n := 0
	for _, p := range paths {
		if r.InScope(p) {
			n++
		}
	}
	return n
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
		si, sj := scopeSpecificity(t.Roles[out[i]], p), scopeSpecificity(t.Roles[out[j]], p)
		if si != sj {
			return si > sj // more specific first
		}
		return out[i] < out[j]
	})
	return out
}

// scopeSpecificity scores how NARROWLY a role's scope covers p: the length of
// the literal (wildcard-free) prefix of its best-matching pattern.
//
// This replaces a count of declared globs, which measured nothing about
// narrowness — a generalist scoped `**` declares one glob and so outranked an
// api specialist scoped to two subtrees, sending work to the wrong role and
// contradicting Owners' own documented promise (dacli 198). A role with no
// declared scope covers everything and scores lowest of all.
func scopeSpecificity(r Role, p string) int {
	if len(r.Scope) == 0 {
		return -1 // no declared scope: maximally general, always last
	}
	best := -1
	for _, g := range r.Scope {
		if !matchGlob(g, p) {
			continue
		}
		if n := literalPrefixLen(g); n > best {
			best = n
		}
	}
	return best
}

// literalPrefixLen is the number of leading characters of a glob before its
// first wildcard — "internal/features/**" scores 18, "**" scores 0, and an
// exact path scores its full length, so an exact match beats every wildcard.
func literalPrefixLen(glob string) int {
	if i := strings.IndexAny(glob, "*?["); i >= 0 {
		return i
	}
	return len(glob)
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
