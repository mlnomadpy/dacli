package team

import "testing"

// Owners promises "the role with the narrowest declared scope wins, since a
// catch-all role should never outrank a specialist" — but it ranked by the
// NUMBER of globs a role declares, which is unrelated to narrowness. A
// generalist scoped `**` (one glob) therefore outranked an api specialist
// scoped to two subtrees, and `dacli team route` sent the work to the wrong
// role (dacli 198).
func TestOwnersRanksBySpecificityNotGlobCount(t *testing.T) {
	tm := &Team{Roles: map[string]Role{
		"generalist": {Name: "generalist", Scope: []string{"**"}},
		"api": {Name: "api", Scope: []string{
			"src/api/**", "src/api2/**", // two globs, but far narrower
		}},
	}}

	got := tm.Owners("src/api/handler.go")
	if len(got) != 2 {
		t.Fatalf("both roles cover the path; got %v", got)
	}
	if got[0] != "api" {
		t.Errorf("owners = %v; the specialist must outrank the catch-all", got)
	}
}

func TestOwnersSpecificityOrdering(t *testing.T) {
	tm := &Team{Roles: map[string]Role{
		"catch-all":  {Name: "catch-all", Scope: []string{"**"}},
		"broad":      {Name: "broad", Scope: []string{"internal/**"}},
		"narrow":     {Name: "narrow", Scope: []string{"internal/features/**"}},
		"narrowest":  {Name: "narrowest", Scope: []string{"internal/features/vcs/**"}},
		"unscoped":   {Name: "unscoped"},
		"unrelated":  {Name: "unrelated", Scope: []string{"docs/**"}},
		"exact-file": {Name: "exact-file", Scope: []string{"internal/features/vcs/vcs.go"}},
	}}

	got := tm.Owners("internal/features/vcs/vcs.go")
	// unrelated must not appear at all; the rest must be narrowest-first.
	want := []string{"exact-file", "narrowest", "narrow", "broad", "catch-all", "unscoped"}
	if len(got) != len(want) {
		t.Fatalf("owners = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("owners = %v, want %v (position %d)", got, want, i)
		}
	}
}

// Equal specificity must break ties deterministically, or `team route` prints a
// different owner run to run.
func TestOwnersTieBreakIsStable(t *testing.T) {
	tm := &Team{Roles: map[string]Role{
		"zeta":  {Name: "zeta", Scope: []string{"src/**"}},
		"alpha": {Name: "alpha", Scope: []string{"src/**"}},
	}}
	for i := 0; i < 20; i++ {
		got := tm.Owners("src/x.go")
		if len(got) != 2 || got[0] != "alpha" || got[1] != "zeta" {
			t.Fatalf("unstable or wrong tie-break: %v", got)
		}
	}
}
