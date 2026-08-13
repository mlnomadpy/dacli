package store

import (
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/team"
)

func TestParseRoleReadsProviderNeutralModelProfile(t *testing.T) {
	d := &mdstore.Doc{}
	d.Front.Set("name", "builder")
	d.Front.Set("model_id", "frontier-medium")
	d.Front.Set("cost_tier", "12")
	d.Front.Set("max_task_points", "8")
	d.Front.Set("context_limit", "200000")
	d.Front.SetList("capability_tags", []string{"code", "vision"})
	r := parseRole(d, "builder")
	if r.Profile.ID != "frontier-medium" || r.Profile.CostTier != 12 || r.Profile.MaxTaskPoints != 8 || r.Profile.ContextLimit != 200000 {
		t.Fatalf("profile was not parsed: %+v", r.Profile)
	}
	if r.Model != r.Profile.ID || r.MaxPoints != r.Profile.MaxTaskPoints {
		t.Fatalf("profile was not projected to migration aliases: model=%q max_points=%g", r.Model, r.MaxPoints)
	}
	if len(r.Profile.CapabilityTags) != 2 || r.Profile.CapabilityTags[1] != "vision" {
		t.Fatalf("capability tags were not parsed: %v", r.Profile.CapabilityTags)
	}
}

func TestParseRoleMigratesExistingModelAndCapacityFields(t *testing.T) {
	d := &mdstore.Doc{}
	d.Front.Set("name", "legacy")
	d.Front.Set("model", "legacy-medium")
	d.Front.Set("max_points", "8")
	r := parseRole(d, "legacy")
	if r.Profile.ID != "legacy-medium" || r.Profile.MaxTaskPoints != 8 {
		t.Fatalf("legacy role did not migrate into a profile: %+v", r.Profile)
	}
}

func TestLegacyRoleFilesPreserveCapacityRouting(t *testing.T) {
	legacy := func(name, modelID, points string) team.Role {
		d := &mdstore.Doc{}
		d.Front.Set("name", name)
		d.Front.Set("role_kind", "implementer")
		d.Front.Set("model", modelID)
		if points != "" {
			d.Front.Set("max_points", points)
		}
		return parseRole(d, name)
	}
	roles := []team.Role{legacy("junior", "legacy-small", "3"), legacy("fixer", "legacy-medium", "8"), legacy("maintainer", "legacy-large", "")}
	for _, tc := range []struct {
		te   float64
		want string
	}{{2, "junior"}, {5, "fixer"}, {12, "maintainer"}} {
		got, ok := team.CheapestCapable(roles, "implementer", tc.te, nil)
		if !ok || got.Name != tc.want {
			t.Errorf("legacy Te %g routed to %q (ok=%v), want %q", tc.te, got.Name, ok, tc.want)
		}
	}
}
