package store

import (
	"fmt"
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
	byName := make(map[string]team.Role, len(roles))
	for _, role := range roles {
		byName[role.Name] = role
	}
	for _, name := range source.FallbackTo {
		candidate, ok := byName[name]
		if !ok || candidate.Runtime == "" || !providerpolicy.EligibleFallback(source.Grant, candidate.Grant, source.Profile.CapabilityTags, candidate.Profile.CapabilityTags) {
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
