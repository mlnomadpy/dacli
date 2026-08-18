package team

import (
	"strings"
	"testing"
)

func TestStrategyExplainsEligibilityAndRanksMeasuredOutcomes(t *testing.T) {
	candidates := []RouteCandidate{
		{Role: Role{Name: "no-tools", Kind: "implementer", Grant: "rw", Runtime: "cheap", Profile: ModelProfile{ID: "small", CostTier: 1, MaxTaskPoints: 8}}, GrantEnforced: true, ContextLimit: 100, RemainingBudget: 1000, CapacityRemaining: 1},
		{Role: Role{Name: "thin-context", Kind: "implementer", Grant: "rw", Runtime: "medium", Profile: ModelProfile{ID: "medium", CostTier: 2, MaxTaskPoints: 8, CapabilityTags: []string{"code"}}}, GrantEnforced: true, ContextLimit: 20, RemainingBudget: 1000, CapacityRemaining: 1},
		{Role: Role{Name: "over-budget", Kind: "implementer", Grant: "rw", Runtime: "dear", Profile: ModelProfile{ID: "large", CostTier: 3, MaxTaskPoints: 8, CapabilityTags: []string{"code"}}}, GrantEnforced: true, ContextLimit: 100, RemainingBudget: 50, CapacityRemaining: 1, Metrics: RouteMetrics{TokensPerCompleted: 100, TokenSamples: 12, FirstPassSuccess: .99, SuccessSamples: 20}},
		{Role: Role{Name: "reliable", Kind: "implementer", Grant: "rw", Runtime: "other", Profile: ModelProfile{ID: "large", CostTier: 3, MaxTaskPoints: 8, CapabilityTags: []string{"code"}}}, GrantEnforced: true, ContextLimit: 100, RemainingBudget: 1000, CapacityRemaining: 1, Metrics: RouteMetrics{TokensPerCompleted: 120, TokenSamples: 15, FirstPassSuccess: .95, SuccessSamples: 20, LatencySeconds: 30}},
		{Role: Role{Name: "unproven", Kind: "implementer", Grant: "rw", Runtime: "fast", Profile: ModelProfile{ID: "large", CostTier: 3, MaxTaskPoints: 8, CapabilityTags: []string{"code"}}}, GrantEnforced: true, ContextLimit: 100, RemainingBudget: 1000, CapacityRemaining: 1, Metrics: RouteMetrics{TokensPerCompleted: 80, TokenSamples: 2, FirstPassSuccess: .5, SuccessSamples: 2, LatencySeconds: 5}},
	}

	explanation := (Strategy{}).Select(RouteRequirements{Kind: "implementer", Grant: "rw", Tools: []string{"code"}, TaskPoints: 5, ContextNeeded: 50, TokenBudget: 200}, candidates)
	if explanation.Selected.Role != "reliable" || explanation.Selected.Runtime != "other" || explanation.Selected.Model != "large" {
		t.Fatalf("selected = %+v, want reliable/other/large", explanation.Selected)
	}
	for name, want := range map[string]string{"no-tools": "tools", "thin-context": "context", "over-budget": "budget"} {
		got := explanation.Candidate(name)
		if got == nil || got.Eligible || !containsReason(got.Exclusions, want) {
			t.Errorf("candidate %s = %+v, want exclusion containing %q", name, got, want)
		}
	}
	got := explanation.Candidate("reliable")
	if got == nil || got.Score.TokenSamples != 15 || got.Score.SuccessSamples != 20 || got.Score.FirstPassSuccess != .95 {
		t.Fatalf("measured score missing sample counts: %+v", got)
	}
}

func TestStrategyHardGatesGrantScopeCapacityQuotaAndProviderPause(t *testing.T) {
	base := Role{Name: "ok", Kind: "implementer", Grant: "rw", Runtime: "rt", Scope: []string{"internal/**"}, Profile: ModelProfile{ID: "m", CostTier: 1, MaxTaskPoints: 5, CapabilityTags: []string{"code"}}}
	tests := []struct {
		name string
		edit func(*RouteCandidate)
		want string
	}{
		{"grant", func(c *RouteCandidate) { c.GrantEnforced = false }, "grant"},
		{"scope", func(c *RouteCandidate) { c.Role.Scope = []string{"docs/**"} }, "scope"},
		{"task capacity", func(c *RouteCandidate) { c.Role.Profile.MaxTaskPoints = 2 }, "task capacity"},
		{"quota", func(c *RouteCandidate) { c.CapacityRemaining = 0 }, "quota"},
		{"provider", func(c *RouteCandidate) { c.ProviderPaused = true }, "provider paused"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			candidate := RouteCandidate{Role: base, GrantEnforced: true, ContextLimit: 100, RemainingBudget: 100, CapacityRemaining: 1}
			tc.edit(&candidate)
			explanation := (Strategy{}).Select(RouteRequirements{Kind: "implementer", Grant: "rw", Tools: []string{"code"}, Paths: []string{"internal/team/routing.go"}, TaskPoints: 4, ContextNeeded: 10, TokenBudget: 10}, []RouteCandidate{candidate})
			got := explanation.Candidate("ok")
			if got == nil || got.Eligible || !containsReason(got.Exclusions, tc.want) || explanation.Selected.Role != "" {
				t.Fatalf("explanation = %+v, want exclusion containing %q and no selection", explanation, tc.want)
			}
		})
	}
}

func containsReason(reasons []string, needle string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, needle) {
			return true
		}
	}
	return false
}

// ModelTier orders models by cost so routing can prefer the cheap one. An
// unknown model must not silently sort as cheapest — that would route real work
// to something nobody priced (dacli 231).
func TestModelTierUsesDeclaredPriceOnly(t *testing.T) {
	for _, tc := range []struct{ declared, want int }{{1, 1}, {98, 98}, {0, 99}, {-1, 99}, {99, 99}, {100, 99}} {
		if got := ModelTier(tc.declared); got != tc.want {
			t.Errorf("ModelTier(%d) = %d, want %d", tc.declared, got, tc.want)
		}
	}
}

func TestCheapestCapableRanksProviderNeutralProfiles(t *testing.T) {
	roles := []Role{
		{Name: "specialist", Kind: "implementer", Runtime: "future-cli", Summary: "implements database migrations", Profile: ModelProfile{ID: "mystery", MaxTaskPoints: 8}},
		{Name: "priced", Kind: "implementer", Runtime: "generic-cli", Profile: ModelProfile{ID: "model-a", CostTier: 7, MaxTaskPoints: 8, ContextLimit: 128000, CapabilityTags: []string{"code", "tools"}}},
	}
	got, ok := CheapestCapableForTitled(roles, "implementer", 5, nil, "Implement database migrations", "")
	if !ok || got.Name != "priced" {
		t.Fatalf("chose %q (ok=%v), want priced profile before unpriced profile", got.Name, ok)
	}
}

// CheapestCapable is the routing rule: the lowest-cost role whose kind fits and
// whose capacity covers the task. The seniority gate already REFUSES a role
// that is too small; this is the other half — actually choosing one, so easy
// work runs on a cheap model without an operator picking by hand.
func TestCheapestCapablePicksTheCheapRoleThatFits(t *testing.T) {
	roles := []Role{
		{Name: "junior", Kind: "implementer", Model: "legacy-small", MaxPoints: 3},
		{Name: "fixer", Kind: "implementer", Model: "legacy-medium", MaxPoints: 8},
		{Name: "maintainer", Kind: "implementer", Model: "legacy-large"}, // uncapped
		{Name: "reviewer", Kind: "reviewer", Model: "legacy-large"},
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
		got, ok := CheapestCapable(roles, "implementer", tc.te, nil)
		if !ok {
			t.Fatalf("Te %g: no role selected (%s)", tc.te, tc.why)
		}
		if got.Name != tc.want {
			t.Errorf("Te %g: chose %s, want %s — %s", tc.te, got.Name, tc.want, tc.why)
		}
	}
}

func TestDeclaredCostTierOutranksSummaryVocabulary(t *testing.T) {
	roles := []Role{
		{Name: "bounded", Kind: "implementer", Summary: "small scoped change", Profile: ModelProfile{ID: "cheap", CostTier: 1, MaxTaskPoints: 3}},
		{Name: "maintainer", Kind: "implementer", Summary: "fix runtime command help and architecture", Profile: ModelProfile{ID: "frontier", CostTier: 3, MaxTaskPoints: 13}},
	}
	pick, ok := CheapestCapableForTitled(roles, "implementer", 2, nil,
		"Fix runtime command help", "update command usage and architecture tests")
	if !ok || pick.Name != "bounded" {
		t.Fatalf("declared tier-1 capable role lost to expensive vocabulary match: %q", pick.Name)
	}
}

// The kind filter is what keeps a reviewer out of an implementation slot.
func TestCheapestCapableHonorsKind(t *testing.T) {
	roles := []Role{
		{Name: "cheap-reviewer", Kind: "reviewer", Model: "legacy-small", MaxPoints: 3},
		{Name: "fixer", Kind: "implementer", Model: "legacy-large"},
	}
	got, ok := CheapestCapable(roles, "implementer", 2, nil)
	if !ok || got.Name != "fixer" {
		t.Fatalf("chose %v (ok=%v); a cheaper role of the WRONG kind must not win", got.Name, ok)
	}
	if _, ok := CheapestCapable(roles, "designer", 1, nil); ok {
		t.Error("no role of that kind exists; selection must fail rather than substitute one")
	}
}

// An unsized task cannot be routed by capacity — saying so beats guessing.
func TestCheapestCapableRefusesWhenNothingFits(t *testing.T) {
	roles := []Role{{Name: "junior", Kind: "implementer", Model: "legacy-small", MaxPoints: 3}}
	if _, ok := CheapestCapable(roles, "implementer", 10, nil); ok {
		t.Error("Te above every cap with no uncapped role must not select anything")
	}
	if _, ok := CheapestCapable(nil, "implementer", 1, nil); ok {
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
		got, _ := CheapestCapable(roles, "implementer", 4, nil)
		if got.Name != "alpha" {
			t.Fatalf("unstable tie-break: got %s", got.Name)
		}
	}
}

// Scope overlap with the task's files must break a cost+capacity tie BEFORE
// name, so a task does not route to a domain-inappropriate role just because
// its name sorts first (dacli 238). Here "alpha" sorts first alphabetically but
// its scope covers none of the task's files, while "zeta" is scoped to exactly
// them — zeta must win.
func TestCheapestCapableBreaksTieByScopeOverlapBeforeName(t *testing.T) {
	roles := []Role{
		{Name: "alpha", Kind: "implementer", Model: "sonnet", MaxPoints: 8, Scope: []string{"web/**"}},
		{Name: "zeta", Kind: "implementer", Model: "sonnet", MaxPoints: 8, Scope: []string{"internal/team/**"}},
	}
	files := []string{"internal/team/routing.go"}
	got, ok := CheapestCapable(roles, "implementer", 4, files)
	if !ok || got.Name != "zeta" {
		t.Fatalf("chose %q (ok=%v), want zeta — the role whose scope covers the task's files must beat the earlier name", got.Name, ok)
	}
	// With no file hints the scope tie-break is a no-op and name still decides,
	// so routing stays deterministic for tasks that mention no paths.
	if got, _ := CheapestCapable(roles, "implementer", 4, nil); got.Name != "alpha" {
		t.Errorf("with no files, chose %q, want alpha — scope must not fire without a signal", got.Name)
	}
}

// A role with NO declared scope must not out-rank a scoped role on overlap: an
// undeclared boundary is generic, not a domain match. Empty-scope "alpha"
// admits every path via InScope, but on a tie it must lose to "zeta" which
// actually named the task's files, not win by sorting first.
func TestCheapestCapableUndeclaredScopeIsNotAMatch(t *testing.T) {
	roles := []Role{
		{Name: "alpha", Kind: "implementer", Model: "sonnet", MaxPoints: 8}, // no scope
		{Name: "zeta", Kind: "implementer", Model: "sonnet", MaxPoints: 8, Scope: []string{"internal/team/**"}},
	}
	got, ok := CheapestCapable(roles, "implementer", 4, []string{"internal/team/routing.go"})
	if !ok || got.Name != "zeta" {
		t.Fatalf("chose %q (ok=%v), want zeta — an undeclared scope is not a domain match", got.Name, ok)
	}
}

// Between two roles on the same model, the TIGHTER cap wins: it is the more
// specialized fit, and it leaves the roomier role free for work that needs it.
func TestCheapestCapablePrefersTheTighterCapAtEqualCost(t *testing.T) {
	roles := []Role{
		{Name: "roomy", Kind: "implementer", Model: "sonnet", MaxPoints: 20},
		{Name: "snug", Kind: "implementer", Model: "sonnet", MaxPoints: 6},
	}
	got, ok := CheapestCapable(roles, "implementer", 5, nil)
	if !ok || got.Name != "snug" {
		t.Errorf("chose %s, want snug — at equal cost the tighter cap is the better fit", got.Name)
	}
}

// "Cheapest capable" has to mean capable FIRST, or the adjective is doing all
// the work. Model price ranked above domain fit, and scope was consulted only
// as a tie-break — so a Go-code audit picked the prompt-auditor (sonnet) over
// the go-auditor (opus). Cheapest, and unable to do the job (task 319).
func TestDomainFitOutranksPrice(t *testing.T) {
	roles := []Role{
		{Name: "prompt-auditor", Kind: "reviewer", Model: "sonnet", MaxPoints: 8,
			Summary: "audit and sharpen the agent prompt registry"},
		{Name: "go-auditor", Kind: "reviewer", Model: "opus", MaxPoints: 8,
			Summary: "audit Go code for performance and best practices"},
	}

	got, ok := CheapestCapableFor(roles, "reviewer", 3.2, nil,
		"Audit the Go packages for swallowed errors")
	if !ok || got.Name != "go-auditor" {
		t.Errorf("a Go audit picked %q; the cheaper role has declared it audits prompts", got.Name)
	}

	// And the cheap specialist still wins its OWN domain — the fix must not
	// simply prefer the expensive role.
	got, ok = CheapestCapableFor(roles, "reviewer", 3.2, nil,
		"Audit the prompt registry for drift")
	if !ok || got.Name != "prompt-auditor" {
		t.Errorf("a prompt audit picked %q, want the prompt-auditor", got.Name)
	}
}

// Shared vocabulary carries no signal: every reviewer summary says "audit", so
// scoring it made both candidates tie and the tie fell through to price. Only
// terms claimed by exactly one candidate discriminate.
func TestSharedSummaryWordsDoNotDiscriminate(t *testing.T) {
	roles := []Role{
		{Name: "cheap", Kind: "reviewer", Model: "haiku", MaxPoints: 8, Summary: "audit things carefully"},
		{Name: "dear", Kind: "reviewer", Model: "opus", MaxPoints: 8, Summary: "audit things thoroughly"},
	}
	// "audit" and "things" are shared; nothing distinctive matches. With no
	// discriminator the ranking must fall back to cheapest — unchanged
	// behaviour for a roster whose summaries say nothing useful.
	got, ok := CheapestCapableFor(roles, "reviewer", 3.2, nil, "audit things")
	if !ok || got.Name != "cheap" {
		t.Errorf("with no distinctive term the cheapest role must win, got %q", got.Name)
	}
}

// A two-letter domain is exactly the kind of term that identifies a specialty,
// and it must match as a WORD — not inside "going" or "algorithm".
func TestShortDomainTermsMatchAsWholeWords(t *testing.T) {
	roles := []Role{
		{Name: "generalist", Kind: "implementer", Model: "haiku", MaxPoints: 8, Summary: "implement anything requested"},
		{Name: "go-dev", Kind: "implementer", Model: "opus", MaxPoints: 8, Summary: "implement Go services"},
	}
	if got, _ := CheapestCapableFor(roles, "implementer", 3.2, nil, "Add a Go handler"); got.Name != "go-dev" {
		t.Errorf("a Go task picked %q, want go-dev", got.Name)
	}
	// "going" must not count as the Go domain.
	if got, _ := CheapestCapableFor(roles, "implementer", 3.2, nil, "Fix the going-away banner"); got.Name != "generalist" {
		t.Errorf("substring match leaked: %q was chosen for a task that never mentions Go", got.Name)
	}
}

// A role's NAME is a declaration of its domain, and the most compressed one
// available. Scoring only the summary threw it away: task 325 ("Trace one
// user-invoked verb end to end across slice SEAMS…") shared no word with
// seam-auditor's summary, so every candidate scored zero and the ranking fell
// through to price — landing on a specialist in a different job entirely.
func TestRoleNameDeclaresDomain(t *testing.T) {
	roles := []Role{
		{Name: "mutation-auditor", Summary: "prove the suite measures what it claims", Kind: "reviewer", Model: "opus", MaxPoints: 8},
		{Name: "seam-auditor", Summary: "audit compositions of individually-correct features", Kind: "reviewer", Model: "opus", MaxPoints: 8},
	}
	pick, ok := CheapestCapableForTitled(roles, "reviewer", 4,
		nil, "Trace one user-invoked verb end to end across slice seams", "")
	if !ok || pick.Name != "seam-auditor" {
		t.Errorf("a task about seams routed to %q; the role NAME states its domain", pick.Name)
	}

	// The shared suffix must stay inert: "auditor" is claimed by both, so
	// distinctiveness already strips it and it cannot decide anything.
	if d := distinctiveTerms(roles); d["auditor"] {
		t.Error(`"auditor" is claimed by both roles and must not count as distinctive`)
	}
}

// "seam" and "seams" are the same domain. Task titles name things in the
// plural while a role declares the bare domain, and whole-word matching missed
// every such pair.
func TestPluralsMatchTheSingularDomain(t *testing.T) {
	for _, tc := range []struct{ declared, used string }{
		{"seam", "seams"}, {"seams", "seam"}, {"prompt", "prompts"},
	} {
		roles := []Role{
			{Name: "generalist", Summary: "review anything at all", Kind: "reviewer", Model: "opus", MaxPoints: 8},
			{Name: "specialist", Summary: "audit the " + tc.declared, Kind: "reviewer", Model: "opus", MaxPoints: 8},
		}
		pick, _ := CheapestCapableForTitled(roles, "reviewer", 4, nil, "Audit the "+tc.used+" carefully", "")
		if pick.Name != "specialist" {
			t.Errorf("declared %q, task said %q: routed to %q", tc.declared, tc.used, pick.Name)
		}
	}

	// Short technical terms must survive the plural rule intact, or a naive
	// strip would turn "js" into "j" and "css" into "cs".
	for _, w := range []string{"js", "css", "aws", "access"} {
		if got := singular(w); got != w {
			t.Errorf("singular(%q) = %q; short and -ss terms must not be truncated", w, got)
		}
	}
}

// The title states what the task IS; the body states how to verify it, in
// generic vocabulary every candidate shares. Scored equally, that vocabulary
// outvoted the one term that identified the domain — 325 tied 2-2 and the tie
// broke alphabetically onto the wrong specialist.
func TestTitleOutweighsBodyVocabulary(t *testing.T) {
	roles := []Role{
		{Name: "mutation-auditor", Summary: "break the code a passing test covers and confirm it fails", Kind: "reviewer", Model: "opus", MaxPoints: 8},
		{Name: "seam-auditor", Summary: "audit compositions of individually-correct features", Kind: "reviewer", Model: "opus", MaxPoints: 8},
	}
	title := "Trace one user-invoked verb end to end across slice seams"
	body := "each handoff records the code path where that assumption fails"

	pick, _ := CheapestCapableForTitled(roles, "reviewer", 4, nil, title, body)
	if pick.Name != "seam-auditor" {
		t.Errorf("body vocabulary outvoted the title's domain term: routed to %q", pick.Name)
	}

	// And the body still counts when the title says nothing — it is secondary,
	// not decorative.
	pick2, _ := CheapestCapableForTitled(roles, "reviewer", 4, nil,
		"Look into the recent regression", "confirm the passing test actually covers it")
	if pick2.Name != "mutation-auditor" {
		t.Errorf("with a silent title the body must still decide: routed to %q", pick2.Name)
	}
}
