package store

import (
	"os"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestProjectLandingRoundTripAndLegacyDefault(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		set  *model.LandingPolicy
		want model.LandingPolicy
	}{
		{"legacy", nil, model.LandingPolicy{}},
		{"configured", &model.LandingPolicy{Mode: model.LandingPR, Base: "develop"}, model.LandingPolicy{Mode: model.LandingPR, Base: "develop"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := CreateProject(w, "a-root", tc.name, tc.name, "g", "")
			if err != nil {
				t.Fatal(err)
			}
			if tc.set != nil {
				if err := ConfigureProjectLanding(p, *tc.set); err != nil {
					t.Fatal(err)
				}
				if err := SaveProject(p); err != nil {
					t.Fatal(err)
				}
			}
			got, err := LoadProject(w, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if got.Landing != tc.want {
				t.Fatalf("landing = %+v, want %+v", got.Landing, tc.want)
			}
		})
	}
}

func TestLoadProjectRejectsInvalidLandingFrontmatter(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ key, value string }{{"landing.mode", "merge"}, {"landing.base", "  "}} {
		p, err := CreateProject(w, "a-root", tc.key, strings.ReplaceAll(tc.key, ".", "-"), "g", "")
		if err != nil {
			t.Fatal(err)
		}
		p.Doc.Front.Set(tc.key, tc.value)
		if err := mdstore.WriteFile(p.Path, p.Doc); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadProject(w, p.Slug); err == nil {
			t.Fatalf("LoadProject accepted %s=%q", tc.key, tc.value)
		}
		if _, err := os.Stat(p.Path); err != nil {
			t.Fatalf("validation unexpectedly removed persisted fixture: %v", err)
		}
	}
}
