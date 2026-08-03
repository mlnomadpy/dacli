package model

import "testing"

// Grant.Exceeds is the attenuation rule: it is what stops a spawned agent from
// handing its own child more capability than it holds. Mutation testing found
// it completely unasserted in this package — flipping `==` to `!=` and `&&` to
// `||` inside it survived the suite. A silently-inverted attenuation check is
// the worst possible failure in the capability model, so it gets an exhaustive
// table rather than an example (dacli 205).
func TestGrantExceedsIsExhaustive(t *testing.T) {
	for _, tc := range []struct {
		child, parent Grant
		want          bool
	}{
		{GrantRW, GrantRO, true},  // the ONLY escalation
		{GrantRW, GrantRW, false}, // equal is not exceeding
		{GrantRO, GrantRW, false}, // narrowing is always fine
		{GrantRO, GrantRO, false},
		{GrantRO, "", false},
		{GrantRW, "", true}, // an unset parent is not rw, so rw exceeds it
		{"", GrantRO, false},
		{"", GrantRW, false},
	} {
		if got := tc.child.Exceeds(tc.parent); got != tc.want {
			t.Errorf("Grant(%q).Exceeds(%q) = %v, want %v", tc.child, tc.parent, got, tc.want)
		}
	}
}

// Risk.Rank is the Impact×Likelihood matrix that decides what must be mitigated
// now versus merely monitored — and rank 1 and 2 are the ones a stage gate can
// require an action plan for. Mutation testing showed the whole matrix was
// unasserted, so an inverted comparison would have silently downgraded real
// risks to "monitor only".
func TestRiskRankMatrix(t *testing.T) {
	for _, tc := range []struct {
		impact, likelihood Level
		want               int
	}{
		{LevelHigh, LevelHigh, 1},   // mitigate immediately
		{LevelHigh, LevelMedium, 2}, // make a plan
		{LevelHigh, LevelLow, 3},    // a low likelihood caps it at monitor
		{LevelLow, LevelHigh, 3},    // ...and so does a low impact
		{LevelLow, LevelLow, 3},
		{LevelMedium, LevelMedium, 2}, // the default arm
		{LevelMedium, LevelHigh, 2},
		{LevelMedium, LevelLow, 3},
	} {
		r := Risk{Impact: tc.impact, Likelihood: tc.likelihood}
		if got := r.Rank(); got != tc.want {
			t.Errorf("Risk{Impact:%s, Likelihood:%s}.Rank() = %d, want %d", tc.impact, tc.likelihood, got, tc.want)
		}
	}
	// The property the gate depends on: only ranks 1 and 2 demand an action
	// plan, and rank is always one of 1, 2, 3.
	for _, i := range []Level{LevelHigh, LevelMedium, LevelLow, ""} {
		for _, l := range []Level{LevelHigh, LevelMedium, LevelLow, ""} {
			if got := (Risk{Impact: i, Likelihood: l}).Rank(); got < 1 || got > 3 {
				t.Errorf("Risk{%s,%s}.Rank() = %d, want 1..3", i, l, got)
			}
		}
	}
}
