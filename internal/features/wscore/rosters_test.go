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

// The default product-building preset is a capability roster, not a provider
// decision. Removing a seat makes the bounded journey incomplete; putting a
// runtime/model on any seat silently changes harness family at init.
func TestAgentsRosterCoversLifecycleAndPreservesHarnessChoice(t *testing.T) {
	roles := rosters["agents"]
	want := map[string]bool{
		"planner": false, "implementer": false, "security-implementer": false,
		"reviewer": false, "security-reviewer": false, "integration-owner": false,
	}
	for _, role := range roles {
		if _, ok := want[role.Name]; ok {
			want[role.Name] = true
		}
		if role.Runtime != "" || role.Model != "" || role.Profile.ID != "" {
			t.Errorf("agents role %s pins runtime/model (%q, %q, %q); roster selection must preserve the configured harness family", role.Name, role.Runtime, role.Model, role.Profile.ID)
		}
	}
	for capability, found := range want {
		if !found {
			t.Errorf("agents roster missing %s capability", capability)
		}
	}
}
