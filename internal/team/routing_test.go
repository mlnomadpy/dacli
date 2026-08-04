package team

import "testing"

// ModelTier orders models by cost so routing can prefer the cheap one. An
// unknown model must not silently sort as cheapest — that would route real work
// to something nobody priced (dacli 231).
func TestModelTier(t *testing.T) {
	if ModelTier("haiku") >= ModelTier("sonnet") {
		t.Error("haiku must rank cheaper than sonnet")
	}
	if ModelTier("sonnet") >= ModelTier("opus") {
		t.Error("sonnet must rank cheaper than opus")
	}
	// Case and vendor prefixes are common in real configs.
	if ModelTier("Sonnet") != ModelTier("sonnet") || ModelTier("claude-3-5-sonnet") != ModelTier("sonnet") {
		t.Error("tier lookup must tolerate case and vendor-prefixed model names")
	}
	// Unknown and unset must be treated as expensive, never as free.
	if ModelTier("some-new-model") <= ModelTier("opus") {
		t.Error("an unknown model must not rank cheaper than the most expensive known one")
	}
	if ModelTier("") <= ModelTier("opus") {
		t.Error("an unset model must not rank cheapest — it is unpriced, not free")
	}
}

// CheapestCapable is the routing rule: the lowest-cost role whose kind fits and
// whose capacity covers the task. The seniority gate already REFUSES a role
// that is too small; this is the other half — actually choosing one, so easy
// work runs on a cheap model without an operator picking by hand.
func TestCheapestCapablePicksTheCheapRoleThatFits(t *testing.T) {
	roles := []Role{
		{Name: "junior", Kind: "implementer", Model: "haiku", MaxPoints: 3},
		{Name: "fixer", Kind: "implementer", Model: "sonnet", MaxPoints: 8},
		{Name: "maintainer", Kind: "implementer", Model: "opus"}, // uncapped
		{Name: "reviewer", Kind: "reviewer", Model: "opus"},
	}

	for _, tc := range []struct {
		te   float64
		want string
		why  string
	}{
		{2, "junior", "small work belongs on the cheapest model that can hold it"},
		{3, "junior", "a cap is inclusive — exactly at capacity still fits"},
		{3.5, "fixer", "just over junior's cap must step up, not squeeze in"},
		{8, "fixer", "at fixer's cap"},
		{9, "maintainer", "above every cap, only the uncapped role remains"},
		{500, "maintainer", "an uncapped role has no ceiling"},
	} {
		got, ok := CheapestCapable(roles, "implementer", tc.te)
		if !ok {
			t.Fatalf("Te %g: no role selected (%s)", tc.te, tc.why)
		}
		if got.Name != tc.want {
			t.Errorf("Te %g: chose %s, want %s — %s", tc.te, got.Name, tc.want, tc.why)
		}
	}
}

// The kind filter is what keeps a reviewer out of an implementation slot.
func TestCheapestCapableHonorsKind(t *testing.T) {
	roles := []Role{
		{Name: "cheap-reviewer", Kind: "reviewer", Model: "haiku", MaxPoints: 3},
		{Name: "fixer", Kind: "implementer", Model: "opus"},
	}
	got, ok := CheapestCapable(roles, "implementer", 2)
	if !ok || got.Name != "fixer" {
		t.Fatalf("chose %v (ok=%v); a cheaper role of the WRONG kind must not win", got.Name, ok)
	}
	if _, ok := CheapestCapable(roles, "designer", 1); ok {
		t.Error("no role of that kind exists; selection must fail rather than substitute one")
	}
}

// An unsized task cannot be routed by capacity — saying so beats guessing.
func TestCheapestCapableRefusesWhenNothingFits(t *testing.T) {
	roles := []Role{{Name: "junior", Kind: "implementer", Model: "haiku", MaxPoints: 3}}
	if _, ok := CheapestCapable(roles, "implementer", 10); ok {
		t.Error("Te above every cap with no uncapped role must not select anything")
	}
	if _, ok := CheapestCapable(nil, "implementer", 1); ok {
		t.Error("an empty roster must not select anything")
	}
}

// Ties must be deterministic, or the same task routes differently run to run.
func TestCheapestCapableTieBreakIsStable(t *testing.T) {
	roles := []Role{
		{Name: "zeta", Kind: "implementer", Model: "sonnet", MaxPoints: 8},
		{Name: "alpha", Kind: "implementer", Model: "sonnet", MaxPoints: 8},
	}
	for i := 0; i < 20; i++ {
		got, _ := CheapestCapable(roles, "implementer", 4)
		if got.Name != "alpha" {
			t.Fatalf("unstable tie-break: got %s", got.Name)
		}
	}
}

// Between two roles on the same model, the TIGHTER cap wins: it is the more
// specialized fit, and it leaves the roomier role free for work that needs it.
func TestCheapestCapablePrefersTheTighterCapAtEqualCost(t *testing.T) {
	roles := []Role{
		{Name: "roomy", Kind: "implementer", Model: "sonnet", MaxPoints: 20},
		{Name: "snug", Kind: "implementer", Model: "sonnet", MaxPoints: 6},
	}
	got, ok := CheapestCapable(roles, "implementer", 5)
	if !ok || got.Name != "snug" {
		t.Errorf("chose %s, want snug — at equal cost the tighter cap is the better fit", got.Name)
	}
}
