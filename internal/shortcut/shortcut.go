// Package shortcut implements named, parameterized command templates.
//
// The token saving is the obvious motivation and the weaker one. An agent
// regenerating `go test ./... -run TestCPM -count=1` costs ~20 tokens; naming
// it costs ~3. Real, but small.
//
// The stronger motivation is that a shortcut is a MEMOIZED DERIVATION. The
// first agent to get a command right paid to discover the correct flags, the
// right working directory, the environment variable the test suite needs. That
// knowledge is normally thrown away the moment the session ends, and the next
// agent rediscovers it — usually getting it subtly wrong first. A shortcut is
// that derivation made durable, reviewable, and attributable.
package shortcut

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Effect classifies what running a shortcut does. It gates execution against
// the caller's capability: a read-only agent may run Read shortcuts freely,
// because reading is how it does its job, but nothing that mutates.
type Effect string

const (
	// EffectRead observes without changing anything: tests, linters, builds
	// to a scratch dir, git log, ls.
	EffectRead Effect = "read"
	// EffectWrite changes the working tree or local state: formatters, code
	// generators, migrations against a dev database.
	EffectWrite Effect = "write"
	// EffectDestructive is irreversible or outward-facing: deploys, pushes,
	// dropping data, publishing. Never runs unattended — see Guard.
	EffectDestructive Effect = "destructive"
)

// Param is a declared template parameter.
type Param struct {
	Name     string `yaml:"name"`
	Summary  string `yaml:"summary,omitempty"`
	Default  string `yaml:"default,omitempty"`
	Required bool   `yaml:"required,omitempty"`

	// Raw disables shell quoting for this parameter. It exists because a few
	// legitimate cases need it (passing a pre-built flag list), and it is
	// declared in the shortcut file rather than at the call site so that
	// enabling it is a reviewable change to a committed file, not something
	// an agent can decide in the moment.
	Raw bool `yaml:"raw,omitempty"`
}

// Shortcut is one named command template.
type Shortcut struct {
	Name    string  `yaml:"name"`
	Summary string  `yaml:"summary"`
	Command string  `yaml:"command"`
	Params  []Param `yaml:"params,omitempty"`
	Effect  Effect  `yaml:"effect"`

	// Roles restricts who may run this. Empty means any role.
	Roles []string `yaml:"roles,omitempty"`

	// Dir is the working directory, relative to the workspace root.
	Dir string `yaml:"dir,omitempty"`

	// Uses counts invocations. It is DERIVED — recomputed from run events at
	// sync, never incremented in place. A directly-incremented counter would
	// make the shortcut file the one object every agent writes concurrently,
	// which is the exact contention the event log exists to prevent.
	//
	// It drives shortcut promotion: dacli watches the event log for repeated
	// ad-hoc commands and suggests turning them into shortcuts, which is
	// where most shortcuts should come from. Asking an agent to predict
	// which commands it will repeat does not work.
	Uses int `yaml:"uses,omitempty"`
}

var (
	ErrUnknownParam  = errors.New("unknown parameter")
	ErrMissingParam  = errors.New("missing required parameter")
	ErrUnclosedGroup = errors.New("unclosed optional group")
	ErrBadTemplate   = errors.New("malformed template")
)

// Expand renders a shortcut into a shell command string.
//
// Every substituted value is single-quoted unless its Param declares Raw.
// This is the security boundary of the whole feature: parameters routinely
// carry model-generated or file-derived text, and a template rendered by
// string concatenation turns `pkg` into an arbitrary-command vector the first
// time a value contains a semicolon. Quoting is not configurable per call.
//
// Template syntax:
//
//	{{name}}      substitute a parameter
//	[[ ... ]]     optional group: dropped entirely if any placeholder
//	              inside it resolves to empty
//
// The optional group exists because a placeholder that resolves to empty
// still renders as an empty quoted argument, and passing a flag an empty
// argument is not the same command as omitting the flag.
func Expand(sc Shortcut, args map[string]string) (string, error) {
	declared := make(map[string]Param, len(sc.Params))
	for _, p := range sc.Params {
		declared[p.Name] = p
	}

	// Reject unknown arguments rather than ignoring them. A silently dropped
	// typo produces a command that runs successfully against the wrong target.
	var unknown []string
	for k := range args {
		if _, ok := declared[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return "", fmt.Errorf("%w: %s", ErrUnknownParam, strings.Join(unknown, ", "))
	}

	values := make(map[string]string, len(sc.Params))
	for _, p := range sc.Params {
		v, given := args[p.Name]
		if !given || v == "" {
			v = p.Default
		}
		if p.Required && v == "" {
			return "", fmt.Errorf("%w: %s", ErrMissingParam, p.Name)
		}
		values[p.Name] = v
	}

	return render(sc.Command, declared, values)
}

func render(tmpl string, declared map[string]Param, values map[string]string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(tmpl); {
		switch {
		case strings.HasPrefix(tmpl[i:], "[["):
			end := strings.Index(tmpl[i+2:], "]]")
			if end < 0 {
				return "", ErrUnclosedGroup
			}
			inner := tmpl[i+2 : i+2+end]
			keep, out, err := renderGroup(inner, declared, values)
			if err != nil {
				return "", err
			}
			if keep {
				b.WriteString(out)
			}
			i += 2 + end + 2

		case strings.HasPrefix(tmpl[i:], "{{"):
			end := strings.Index(tmpl[i+2:], "}}")
			if end < 0 {
				return "", fmt.Errorf("%w: unclosed placeholder", ErrBadTemplate)
			}
			name := strings.TrimSpace(tmpl[i+2 : i+2+end])
			p, ok := declared[name]
			if !ok {
				return "", fmt.Errorf("%w: %s is used in the command but not declared", ErrUnknownParam, name)
			}
			b.WriteString(substitute(p, values[name]))
			i += 2 + end + 2

		default:
			b.WriteByte(tmpl[i])
			i++
		}
	}
	return strings.TrimSpace(collapseSpaces(b.String())), nil
}

// renderGroup renders an optional group, reporting whether to keep it.
func renderGroup(inner string, declared map[string]Param, values map[string]string) (bool, string, error) {
	for _, name := range placeholders(inner) {
		if _, ok := declared[name]; !ok {
			return false, "", fmt.Errorf("%w: %s is used in the command but not declared", ErrUnknownParam, name)
		}
		if values[name] == "" {
			return false, "", nil
		}
	}
	out, err := render(inner, declared, values)
	if err != nil {
		return false, "", err
	}
	return true, " " + out + " ", nil
}

func placeholders(s string) []string {
	var out []string
	for {
		i := strings.Index(s, "{{")
		if i < 0 {
			return out
		}
		j := strings.Index(s[i+2:], "}}")
		if j < 0 {
			return out
		}
		out = append(out, strings.TrimSpace(s[i+2:i+2+j]))
		s = s[i+2+j+2:]
	}
}

func substitute(p Param, v string) string {
	if p.Raw {
		return v
	}
	return Quote(v)
}

// Quote wraps s in POSIX single quotes, which disable every form of shell
// interpretation. Embedded single quotes are closed, backslash-escaped, and
// reopened, which is the only correct way to nest a quote inside a quoted
// POSIX string.
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	if !needsQuoting(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// needsQuoting reports whether s contains anything a shell would interpret.
// The allowed set is deliberately conservative: when in doubt, quote.
func needsQuoting(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '_', c == '-', c == '.', c == '/', c == ':', c == '=', c == '@', c == ',', c == '+':
		default:
			return true
		}
	}
	return false
}

func collapseSpaces(s string) string {
	var b strings.Builder
	space := false
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteByte(s[i])
	}
	return b.String()
}

// Guard reports whether a caller may run this shortcut.
//
// Two independent gates, both of which must pass: the capability gate (can
// this agent mutate anything at all) and the role gate (is this shortcut part
// of this role's toolkit). They are separate because they answer different
// questions — a backend agent with write capability still has no business
// running the frontend deploy.
func Guard(sc Shortcut, role string, canMutate bool, confirmed bool) error {
	if len(sc.Roles) > 0 && role != "" {
		var allowed bool
		for _, r := range sc.Roles {
			if r == role {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("shortcut %q is not in role %q's toolkit (allowed: %s)",
				sc.Name, role, strings.Join(sc.Roles, ", "))
		}
	}

	switch sc.Effect {
	case EffectRead:
		return nil
	case EffectWrite:
		if !canMutate {
			return fmt.Errorf("shortcut %q has write effect and this agent holds a read-only grant", sc.Name)
		}
		return nil
	case EffectDestructive:
		if !canMutate {
			return fmt.Errorf("shortcut %q is destructive and this agent holds a read-only grant", sc.Name)
		}
		if !confirmed {
			// Destructive shortcuts are exactly the commands an agent should
			// not be able to reach by autocompleting a name. Requiring an
			// explicit confirmation keeps "deploy" from being one token away
			// from "test" in the shortcut list.
			return fmt.Errorf("shortcut %q is destructive and requires explicit confirmation", sc.Name)
		}
		return nil
	default:
		return fmt.Errorf("shortcut %q has no declared effect; refusing to run", sc.Name)
	}
}

// Catalog renders the shortcut list for injection into a context brief.
//
// This is where the token argument actually gets settled. A catalog of N
// shortcuts costs roughly N x 12 tokens in EVERY brief, and saves ~20 tokens
// per invocation. A shortcut nobody calls is a permanent tax, so the catalog
// is ordered by use count and truncated — an unused shortcut still exists and
// still runs, it just stops being advertised.
func Catalog(scs []Shortcut, role string, limit int) string {
	var vis []Shortcut
	for _, sc := range scs {
		if len(sc.Roles) > 0 && role != "" {
			var ok bool
			for _, r := range sc.Roles {
				if r == role {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		vis = append(vis, sc)
	}
	sort.SliceStable(vis, func(i, j int) bool {
		if vis[i].Uses != vis[j].Uses {
			return vis[i].Uses > vis[j].Uses
		}
		return vis[i].Name < vis[j].Name
	})

	dropped := 0
	if limit > 0 && len(vis) > limit {
		dropped = len(vis) - limit
		vis = vis[:limit]
	}

	var b strings.Builder
	for _, sc := range vis {
		fmt.Fprintf(&b, "- `dacli run %s` — %s", sc.Name, sc.Summary)
		if sc.Effect != EffectRead {
			fmt.Fprintf(&b, " (%s)", sc.Effect)
		}
		b.WriteByte('\n')
	}
	if dropped > 0 {
		fmt.Fprintf(&b, "<!-- dacli: %d rarely-used shortcuts omitted; `dacli run --list` shows all -->\n", dropped)
	}
	return b.String()
}
