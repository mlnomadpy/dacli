package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mlnomadpy/dacli/internal/providerpolicy"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// RuntimeLimits is the persistence boundary for provider cooldowns. Its files
// live in the gitignored run record tree, so they survive loop process restarts
// without becoming portable claims about another machine's provider account.
type RuntimeLimits struct {
	breaker providerpolicy.Breaker
}

// Report writes the exact same transition to the operator stream and durable
// runtime-policy log. Callers cannot accidentally print one account of a
// fallback while recording another.
func (l RuntimeLimits) Report(out io.Writer, transition providerpolicy.Transition) error {
	line := transition.String()
	if _, err := fmt.Fprintln(out, line); err != nil {
		return err
	}
	if err := os.MkdirAll(l.breaker.Dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(l.breaker.Dir, "transitions.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintln(f, line); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func LoadRuntimeLimits(w *workspace.Workspace) RuntimeLimits {
	return RuntimeLimits{breaker: providerpolicy.Breaker{Dir: filepath.Join(w.RunsDir(), "runtime-cooldowns")}}
}

func (l RuntimeLimits) Open(runtime string) (providerpolicy.Cooldown, bool, error) {
	return l.breaker.Open(runtime)
}

func (l RuntimeLimits) Record(runtime string, outcome providerpolicy.Outcome, cooldown time.Duration) (providerpolicy.Cooldown, error) {
	return l.breaker.Record(runtime, outcome, cooldown)
}

// SelectFallback walks only the source role's declared order. It never scans
// the roster for an attractive substitute, and rejects candidates that weaken
// grant or capability requirements.
func SelectFallback(source team.Role, roles []team.Role, limits RuntimeLimits) (team.Role, providerpolicy.Cooldown, bool, error) {
	return SelectFallbackMatching(source, roles, limits, nil)
}

// SelectFallbackMatching adds an authority boundary to the normal capability
// checks. A nil predicate preserves the historical behavior; orchestration can
// use it to keep fallback inside an explicit harness allowlist.
func SelectFallbackMatching(source team.Role, roles []team.Role, limits RuntimeLimits, allowed func(team.Role) bool) (team.Role, providerpolicy.Cooldown, bool, error) {
	byName := make(map[string]team.Role, len(roles))
	for _, role := range roles {
		byName[role.Name] = role
	}
	for _, name := range source.FallbackTo {
		candidate, ok := byName[name]
		if !ok || candidate.Runtime == "" || (allowed != nil && !allowed(candidate)) || !providerpolicy.EligibleFallback(source.Grant, candidate.Grant, source.Profile.CapabilityTags, candidate.Profile.CapabilityTags) {
			continue
		}
		cool, open, err := limits.Open(candidate.Runtime)
		if err != nil {
			return team.Role{}, providerpolicy.Cooldown{}, false, fmt.Errorf("fallback runtime %s: %w", candidate.Runtime, err)
		}
		if !open {
			return candidate, cool, true, nil
		}
	}
	return team.Role{}, providerpolicy.Cooldown{}, false, nil
}
