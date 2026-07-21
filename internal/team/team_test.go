package team

import (
	"errors"
	"testing"
)

func testTeam(t *testing.T) *Team {
	t.Helper()
	tm, err := New([]Role{
		{
			Name:       "frontend",
			Scope:      []string{"web/**", "*.css"},
			Shortcuts:  []string{"build-web"},
			EscalateTo: []string{"backend", "architect"},
			WIP:        2,
		},
		{
			Name:       "backend",
			Scope:      []string{"internal/**", "cmd/**"},
			OutOfScope: []string{"internal/legacy/**"},
			EscalateTo: []string{"architect"},
		},
		{
			Name:       "architect",
			EscalateTo: []string{Human},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func TestInScopeGlobs(t *testing.T) {
	tm := testTeam(t)
	be := tm.Roles["backend"]

	for _, p := range []string{"internal/spm/estimate.go", "internal/a/b/c/d.go", "cmd/dacli/main.go"} {
		if !be.InScope(p) {
			t.Errorf("%q should be in backend scope", p)
		}
	}
	for _, p := range []string{"web/app.tsx", "README.md"} {
		if be.InScope(p) {
			t.Errorf("%q should not be in backend scope", p)
		}
	}
}

// A deny that a broader allow can override is not a boundary.
func TestOutOfScopeBeatsScope(t *testing.T) {
	be := testTeam(t).Roles["backend"]
	if be.InScope("internal/legacy/old.go") {
		t.Error("out_of_scope must win over a matching scope glob")
	}
}

func TestEmptyScopeIsPermissive(t *testing.T) {
	arch := testTeam(t).Roles["architect"]
	if !arch.InScope("anything/at/all.go") {
		t.Error("a role with no declared scope should have no fence")
	}
}

func TestSingleSegmentGlobDoesNotCrossDirectories(t *testing.T) {
	fe := testTeam(t).Roles["frontend"]
	if !fe.InScope("main.css") {
		t.Error("*.css should match a top-level file")
	}
	if fe.InScope("web/deep/main.css") == false {
		t.Error("web/** should still match nested files")
	}
	r := Role{Scope: []string{"src/*.go"}}
	if r.InScope("src/nested/x.go") {
		t.Error("a single * must not cross a directory separator")
	}
}

func TestRouteReturnsSelfWhenInScope(t *testing.T) {
	chain, err := testTeam(t).Route("backend", "internal/spm/x.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 1 || chain[0] != "backend" {
		t.Errorf("chain = %v, want [backend]", chain)
	}
}

func TestRouteFollowsEscalationChain(t *testing.T) {
	chain, err := testTeam(t).Route("frontend", "internal/spm/x.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 2 || chain[0] != "frontend" || chain[1] != "backend" {
		t.Errorf("chain = %v, want [frontend backend]", chain)
	}
}

// Falling off the end of the tree is a normal outcome. A tree that can never
// say "nobody here owns this" will instead have somebody guess, and the guess
// ships.
func TestRouteEscalatesOutWhenNothingCovers(t *testing.T) {
	tm, err := New([]Role{
		{Name: "frontend", Scope: []string{"web/**"}, EscalateTo: []string{"backend"}},
		{Name: "backend", Scope: []string{"internal/**"}, EscalateTo: []string{Human}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tm.Route("frontend", "infra/terraform/main.tf"); !errors.Is(err, ErrNoOwner) {
		t.Errorf("err = %v, want ErrNoOwner", err)
	}
}

// A mutual escalate_to pair is a configuration mistake, not an infinite loop.
func TestRouteHandlesCycles(t *testing.T) {
	tm, err := New([]Role{
		{Name: "a", Scope: []string{"a/**"}, EscalateTo: []string{"b"}},
		{Name: "b", Scope: []string{"b/**"}, EscalateTo: []string{"a"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tm.Route("a", "c/x.go"); !errors.Is(err, ErrNoOwner) {
		t.Errorf("err = %v, want ErrNoOwner", err)
	}
}

func TestRouteRejectsUnknownRoles(t *testing.T) {
	tm := testTeam(t)
	if _, err := tm.Route("ghost", "x.go"); err == nil {
		t.Error("unknown starting role should error")
	}
}

// A catch-all role must never outrank a specialist.
func TestOwnersPrefersNarrowestScope(t *testing.T) {
	owners := testTeam(t).Owners("internal/spm/x.go")
	if len(owners) == 0 {
		t.Fatal("no owners found")
	}
	if owners[0] != "backend" {
		t.Errorf("owners = %v, want backend first (architect has no fence)", owners)
	}
}

func TestCanRun(t *testing.T) {
	tm := testTeam(t)
	if !tm.Roles["frontend"].CanRun("build-web") {
		t.Error("frontend should run its declared shortcut")
	}
	if tm.Roles["frontend"].CanRun("deploy-db") {
		t.Error("frontend should not run an undeclared shortcut")
	}
	if !tm.Roles["backend"].CanRun("anything") {
		t.Error("a role with no declared shortcuts should be unrestricted")
	}
}

// Burning Across, made preventable rather than merely detectable.
func TestWIPLimit(t *testing.T) {
	tm := testTeam(t)
	if tm.WIPExceeded("frontend", 1) {
		t.Error("1 active against a WIP of 2 should be allowed")
	}
	if !tm.WIPExceeded("frontend", 2) {
		t.Error("2 active against a WIP of 2 should block a third spawn")
	}
	if tm.WIPExceeded("backend", 99) {
		t.Error("a role with no WIP limit should never block")
	}
}

func TestNewRejectsDuplicateAndUnnamedRoles(t *testing.T) {
	if _, err := New([]Role{{Name: "a"}, {Name: "a"}}); err == nil {
		t.Error("duplicate role names should be rejected")
	}
	if _, err := New([]Role{{}}); err == nil {
		t.Error("an unnamed role should be rejected")
	}
}
