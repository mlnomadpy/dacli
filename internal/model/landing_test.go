package model

import "testing"

func TestResolveLandingPrecedence(t *testing.T) {
	pr := LandingPR
	release := "release"
	tests := []struct {
		name     string
		config   LandingPolicy
		override LandingOverride
		want     LandingPolicy
		explicit bool
	}{
		{"legacy default", LandingPolicy{}, LandingOverride{}, LandingPolicy{Mode: LandingLocal}, false},
		{"configured", LandingPolicy{Mode: LandingPR, Base: "main"}, LandingOverride{}, LandingPolicy{Mode: LandingPR, Base: "main"}, false},
		{"cli mode", LandingPolicy{Mode: LandingLocal, Base: "main"}, LandingOverride{Mode: &pr}, LandingPolicy{Mode: LandingPR, Base: "main"}, true},
		{"cli base", LandingPolicy{Mode: LandingLocal, Base: "main"}, LandingOverride{Base: &release}, LandingPolicy{Mode: LandingLocal, Base: "release"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, explicit, err := ResolveLanding(tt.config, tt.override)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want || explicit != tt.explicit {
				t.Fatalf("got (%+v, %t), want (%+v, %t)", got, explicit, tt.want, tt.explicit)
			}
		})
	}
}

func TestResolveLandingRejectsInvalidValues(t *testing.T) {
	badMode := LandingMode("merge")
	blank := "  "
	for _, override := range []LandingOverride{{Mode: &badMode}, {Base: &blank}} {
		if _, _, err := ResolveLanding(LandingPolicy{}, override); err == nil {
			t.Fatalf("ResolveLanding accepted invalid override %+v", override)
		}
	}
}
