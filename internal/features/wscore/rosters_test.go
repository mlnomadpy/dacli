package wscore

import (
	"strings"
	"testing"
)

// Every seeded role must change what an agent can DO — scope, grant, or an
// escalation edge — never just what it calls itself (docs/TEAM.md § 2). A
// roster with a costume role would defeat the point of seeding one.
func TestRostersEveryRoleCarvesABoundary(t *testing.T) {
	for rosterName, roles := range rosters {
		if len(roles) == 0 {
			t.Errorf("roster %q has no roles", rosterName)
		}
		for _, r := range roles {
			if r.Name == "" {
				t.Errorf("roster %q has an unnamed role", rosterName)
			}
			mechanical := len(r.Scope) > 0 || len(r.Skills) > 0 || len(r.EscalateTo) > 0 ||
				r.Grant != "" || r.WIP != 0
			if !mechanical {
				t.Errorf("roster %q role %q carves no boundary (no scope/skills/escalate_to/grant/wip)", rosterName, r.Name)
			}
		}
	}
}

func TestRosterNamesListsEveryRoster(t *testing.T) {
	got := strings.Split(rosterNames(), ", ")
	for name := range rosters {
		found := false
		for _, g := range got {
			if g == name {
				found = true
			}
		}
		if !found {
			t.Errorf("rosterNames() missing %q: %v", name, got)
		}
	}
}
