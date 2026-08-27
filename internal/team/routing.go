package team

import (
	"fmt"
	"sort"
	"strings"
)

// Strategy composes routing policy without coupling it to a provider or CLI.
// Eligibility is evaluated before ranking, and every fact used in either
// phase is retained in the returned Explanation.
type Strategy struct{}

type RouteRequirements struct {
	Kind, Grant   string
	Title, Body   string
	Tools, Paths  []string
	TaskPoints    float64
	ContextNeeded int
	TokenBudget   float64
	// RequireTokenCeiling makes runtime support for a hard per-run token cap
	// an eligibility boundary. The scheduler consumes the adapter capability;
	// it never needs to know which provider or CLI the adapter represents.
	RequireTokenCeiling bool
}

type RouteMetrics struct {
	TokensPerCompleted float64 `json:"tokens_per_completed"`
	TokenSamples       int     `json:"token_samples"`
	FirstPassSuccess   float64 `json:"first_pass_success"`
	SuccessSamples     int     `json:"success_samples"`
	LatencySeconds     float64 `json:"latency_seconds"`
}

type RouteCandidate struct {
	Role                 Role
	GrantEnforced        bool
	ContextLimit         int
	CapacityRemaining    int
	RemainingBudget      float64
	ProviderPaused       bool
	TokenCeilingEnforced bool
	Metrics              RouteMetrics
}

type RouteScore struct {
	CostTier           int     `json:"cost_tier"`
	TokensPerCompleted float64 `json:"tokens_per_completed"`
	TokenSamples       int     `json:"token_samples"`
	FirstPassSuccess   float64 `json:"first_pass_success"`
	SuccessSamples     int     `json:"success_samples"`
	LatencySeconds     float64 `json:"latency_seconds"`
	DomainFit          int     `json:"domain_fit"`
	Total              float64 `json:"total"`
}

type CandidateExplanation struct {
	Role       string     `json:"role"`
	Runtime    string     `json:"runtime"`
	Model      string     `json:"model"`
	Eligible   bool       `json:"eligible"`
	Exclusions []string   `json:"exclusions,omitempty"`
	Score      RouteScore `json:"score"`
}

type RouteSelection struct {
	Role, Runtime, Model string
}

type Explanation struct {
	Candidates []CandidateExplanation `json:"candidates"`
	Selected   RouteSelection         `json:"selected"`
	Source     string                 `json:"source,omitempty"`
	Uplift     string                 `json:"consequence_uplift,omitempty"`
}

func (e Explanation) Candidate(name string) *CandidateExplanation {
	for i := range e.Candidates {
		if e.Candidates[i].Role == name {
			return &e.Candidates[i]
		}
	}
	return nil
}

// Select applies hard gates to every candidate before comparing scores. A
// zero sample count is retained as unknown evidence, never presented as a
// measured zero. Thin histories receive a confidence penalty until n>=10.
func (Strategy) Select(req RouteRequirements, candidates []RouteCandidate) Explanation {
	explanation := Explanation{Source: "cheapest capable by cost, capacity, kind, scope, grant, and live availability"}
	for _, candidate := range candidates {
		r := candidate.Role
		item := CandidateExplanation{Role: r.Name, Runtime: r.Runtime, Model: r.ModelID(), Eligible: true}
		exclude := func(reason string) { item.Eligible = false; item.Exclusions = append(item.Exclusions, reason) }
		if !strings.EqualFold(r.Kind, req.Kind) {
			exclude(fmt.Sprintf("role kind %q does not satisfy %q", r.Kind, req.Kind))
		}
		if !grantAtLeast(r.Grant, req.Grant) || !candidate.GrantEnforced {
			exclude("grant is not enforceable at the required strength")
		}
		for _, tool := range req.Tools {
			if !hasFold(r.Profile.CapabilityTags, tool) {
				exclude("missing required tools: " + tool)
			}
		}
		for _, path := range req.Paths {
			if !r.InScope(path) {
				exclude("scope excludes " + path)
				break
			}
		}
		if cap := r.TaskCapacity(); cap > 0 && req.TaskPoints > cap {
			exclude(fmt.Sprintf("task capacity %.1f is below %.1f", cap, req.TaskPoints))
		}
		if candidate.CapacityRemaining <= 0 {
			exclude("quota or concurrency capacity exhausted")
		}
		if req.ContextNeeded > 0 && (candidate.ContextLimit <= 0 || candidate.ContextLimit < req.ContextNeeded) {
			exclude("context limit is below task requirement")
		}
		if candidate.ProviderPaused {
			exclude("provider paused by rolling budget")
		}
		if req.RequireTokenCeiling && !candidate.TokenCeilingEnforced {
			exclude("runtime cannot enforce the required token ceiling")
		}
		expected := candidate.Metrics.TokensPerCompleted
		if expected > 0 && ((candidate.RemainingBudget > 0 && expected > candidate.RemainingBudget) || (req.TokenBudget > 0 && expected > req.TokenBudget)) {
			exclude("remaining budget is below calibrated task cost")
		}

		item.Score = scoreCandidate(candidate, req, candidates)
		explanation.Candidates = append(explanation.Candidates, item)
	}
	sort.SliceStable(explanation.Candidates, func(i, j int) bool {
		a, b := explanation.Candidates[i], explanation.Candidates[j]
		if a.Eligible != b.Eligible {
			return a.Eligible
		}
		if a.Score.Total != b.Score.Total {
			return a.Score.Total < b.Score.Total
		}
		return a.Role < b.Role
	})
	if len(explanation.Candidates) > 0 && explanation.Candidates[0].Eligible {
		pick := explanation.Candidates[0]
		explanation.Selected = RouteSelection{Role: pick.Role, Runtime: pick.Runtime, Model: pick.Model}
		if HighConsequence(req.Title, req.Paths) {
			selectedTier := pick.Score.CostTier
			for _, candidate := range explanation.Candidates {
				if candidate.Eligible && candidate.Score.CostTier > selectedTier && candidate.Score.CostTier < tierUnknown && upliftCompatible(pick, candidate, req, candidates) {
					explanation.Selected = RouteSelection{Role: candidate.Role, Runtime: candidate.Runtime, Model: candidate.Model}
					explanation.Uplift = fmt.Sprintf("high-consequence work uplifted from %s (tier %d) to %s (tier %d)", pick.Role, selectedTier, candidate.Role, candidate.Score.CostTier)
					explanation.Source = "consequence-aware uplift after cheapest-capable eligibility"
					break
				}
			}
		}
	}
	return explanation
}

// HighConsequence is shared by interactive assignment and unattended loops.
// Keeping this vocabulary beside Strategy prevents a preview from promising
// an uplift that execution silently omits (task 495).
func HighConsequence(title string, paths []string) bool {
	s := strings.NewReplacer("/", " ", "_", " ", "-", " ").Replace(title + " " + strings.Join(paths, " "))
	words := taskWords(s)
	for _, word := range []string{"security", "auth", "authentication", "authorization", "permission", "migration", "transaction", "transactional", "persistence", "database", "sql", "lease", "recovery", "destructive", "ambiguous"} {
		if words[word] {
			return true
		}
	}
	return false
}

// upliftCompatible prevents consequence policy from spending a stronger model
// on an unrelated specialty merely because that role is the next price tier.
// A stronger candidate must improve domain fit, cover a concrete task path, or
// declare itself general-purpose. The ordinary eligibility gates still apply.
func upliftCompatible(from, to CandidateExplanation, req RouteRequirements, candidates []RouteCandidate) bool {
	var role Role
	for _, candidate := range candidates {
		if candidate.Role.Name == to.Role {
			role = candidate.Role
			break
		}
	}
	if to.Score.DomainFit > from.Score.DomainFit || (len(req.Paths) > 0 && role.ScopeOverlap(req.Paths) > 0) {
		return true
	}
	if len(role.Scope) == 0 {
		return true
	}
	for _, scope := range role.Scope {
		if scope == "**" || scope == "**/*" {
			return true
		}
	}
	return false
}

func scoreCandidate(candidate RouteCandidate, req RouteRequirements, all []RouteCandidate) RouteScore {
	r := candidate.Role
	roles := make([]Role, 0, len(all))
	for _, c := range all {
		roles = append(roles, c.Role)
	}
	fit := relevanceOf(r, req.Title, req.Body, req.Paths, distinctiveTerms(roles))
	m := candidate.Metrics
	confidencePenalty := 0.0
	if m.TokenSamples < 10 {
		confidencePenalty += float64(10-m.TokenSamples) * 10
	}
	if m.SuccessSamples < 10 {
		confidencePenalty += float64(10-m.SuccessSamples) * 10
	}
	total := float64(ModelTier(r.Profile.CostTier))*100 + m.TokensPerCompleted/100 + m.LatencySeconds/60 - m.FirstPassSuccess*50 - float64(fit)*10 + confidencePenalty
	return RouteScore{CostTier: ModelTier(r.Profile.CostTier), TokensPerCompleted: m.TokensPerCompleted, TokenSamples: m.TokenSamples, FirstPassSuccess: m.FirstPassSuccess, SuccessSamples: m.SuccessSamples, LatencySeconds: m.LatencySeconds, DomainFit: fit, Total: total}
}

func grantAtLeast(have, need string) bool {
	strength := map[string]int{"ro": 1, "rw": 2}
	return strength[strings.ToLower(have)] >= strength[strings.ToLower(need)] && strength[strings.ToLower(need)] > 0
}

func hasFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

// tierUnknown sorts after every known model.
const tierUnknown = 99

// ModelTier validates a declared provider-neutral ordering rank. Unknown,
// unpriced, and reserved values rank last rather than becoming accidentally
// cheap. Model identifiers never participate in pricing.
func ModelTier(declared int) int {
	if declared < 1 || declared > 98 {
		return tierUnknown
	}
	return declared
}

// CheapestCapable returns the lowest-cost role that may do work of the given
// kind at size te, or ok=false when no role in the roster qualifies.
//
// This is the missing half of the seniority gate. That gate already REFUSES a
// role whose MaxPoints is below the task's Te — "assign a heavier role, or
// decompose the task" — but nothing ever chose one, so model tiering existed
// only as a per-role setting an operator applied by hand. With capacities
// declared, selection becomes mechanical: small work lands on the cheap model
// because the cheap role can hold it, and expensive models are spent on the
// work that actually needs them.
//
// Ordering: cheapest model first; at equal cost the TIGHTER cap wins, because
// it is the more specialized fit and leaves the roomier role free for work that
// needs the room; then the role whose declared Scope covers MORE of the task's
// files, so a cost+capacity tie does not route domain-inappropriate work to
// whichever role's name sorts first (dacli 238); then by name, so the same task
// always routes the same way. MaxPoints <= 0 means uncapped and is treated as
// the loosest possible fit. files may be nil, in which case the scope tie-break
// is a no-op and ordering is unchanged.
func CheapestCapable(roles []Role, kind string, te float64, files []string) (Role, bool) {
	return CheapestCapableFor(roles, kind, te, files, "")
}

// CheapestCapableFor is CheapestCapable with the task's TEXT, so domain fit can
// be judged when the task names no files — which is the common case for audit
// and research work, and exactly where the old ranking went wrong.
//
// Callers that can separate the title from the body should prefer
// CheapestCapableForTitled: the title is by far the higher-signal half, and
// blending the two loses that.
func CheapestCapableFor(roles []Role, kind string, te float64, files []string, taskText string) (Role, bool) {
	return CheapestCapableForTitled(roles, kind, te, files, taskText, "")
}

// CheapestCapableForTitled ranks with the task's title weighted above its body.
//
// The title states what the task IS; the body states how to verify it, in
// generic engineering vocabulary every candidate shares. Scoring them equally
// let that vocabulary outvote the one term that actually identified the
// domain: task 325 ("Trace one user-invoked verb end to end across slice
// SEAMS…") tied seam-auditor against mutation-auditor at 2 apiece — seam's
// point came from the title, mutation's from incidental words in the
// acceptance criteria — and the tie fell through to alphabetical order, which
// picked the wrong specialist. Same failure as scoring shared summary
// vocabulary (task 319), one level down: more text is not more signal.
func CheapestCapableForTitled(roles []Role, kind string, te float64, files []string, title, body string) (Role, bool) {
	var fit []Role
	for _, r := range roles {
		if !strings.EqualFold(r.Kind, kind) {
			continue
		}
		if cap := r.TaskCapacity(); cap > 0 && te > cap {
			continue // the seniority gate would refuse this spawn
		}
		fit = append(fit, r)
	}
	if len(fit) == 0 {
		return Role{}, false
	}
	// Domain fit outranks price — but ONLY when it discriminates. If no
	// candidate declares any relevance to this work (a generic task, or a
	// roster whose summaries say nothing useful), every score is 0 and the
	// ranking falls through to cheapest-capable exactly as before. So this
	// cannot make ordinary routing more expensive; it only stops a cheap role
	// winning work it has declared it does not do.
	distinctive := distinctiveTerms(fit)
	rel := make(map[string]int, len(fit))
	discriminates := false
	for _, r := range fit {
		rel[r.Name] = relevanceOf(r, title, body, files, distinctive)
		if rel[r.Name] > 0 {
			discriminates = true
		}
	}

	sort.SliceStable(fit, func(i, j int) bool {
		// A declared price always beats an unknown one, and the cheapest
		// declared tier is the routing floor promised by team assign. Domain
		// relevance chooses within a tier; it must not silently promote a
		// frontier generalist over a capable cheap role merely because the
		// expensive summary repeats more words from the task (issue #689).
		ti, tj := ModelTier(fit[i].Profile.CostTier), ModelTier(fit[j].Profile.CostTier)
		if (ti == tierUnknown) != (tj == tierUnknown) {
			return ti != tierUnknown
		}
		if ti != tj {
			return ti < tj
		}
		if discriminates {
			if ri, rj := rel[fit[i].Name], rel[fit[j].Name]; ri != rj {
				return ri > rj
			}
		}
		ci, cj := capacityRank(fit[i]), capacityRank(fit[j])
		if ci != cj {
			return ci < cj
		}
		// Scope overlap breaks the tie before name: the role whose declared
		// boundary covers more of the task's files is the domain-appropriate
		// fit, so it wins over one that merely sorts earlier alphabetically.
		oi, oj := fit[i].ScopeOverlap(files), fit[j].ScopeOverlap(files)
		if oi != oj {
			return oi > oj
		}
		return fit[i].Name < fit[j].Name
	})
	return fit[0], true
}

// capacityRank orders roles from tightest cap to loosest, with uncapped last.
func capacityRank(r Role) float64 {
	if r.TaskCapacity() <= 0 {
		return 1 << 30
	}
	return r.TaskCapacity()
}

// relevanceOf scores how well one role's declared domain matches the work,
// counting only terms DISTINCTIVE to it among the candidates.
//
// Distinctiveness is the whole mechanism. Every reviewer summary contains
// "audit", so matching it tells you nothing — scoring shared vocabulary made
// a Go audit and a prompt audit tie, and the tie fell through to price, which
// is how the cheapest role won work it had declared it does not do (task 319).
// The words that identify a domain are the ones only one candidate claims:
// "go", "prompt", "registry", "vue".
//
// Three signals, because any one is often unavailable. Scope globs answer when
// the task names files; when it names none — as every audit task filed this
// session did — the role's SUMMARY declares what it does, and its NAME does
// the same in the most compressed form available.
//
// The name is not a tie-break dressed up as a signal. An operator who calls a
// role `seam-auditor` has stated its domain as plainly as a name can, and
// scoring only the summary threw that away: task 325 ("Trace one user-invoked
// verb end to end across slice SEAMS…") shared no word with seam-auditor's
// summary, scored zero against every candidate, and fell through to price —
// landing on mutation-auditor, whose charter is a different job entirely. The
// shared suffixes cost nothing because distinctiveness already filters them:
// "auditor" is claimed by four roles and carries no signal, while "seam" and
// "mutation" are claimed by exactly one each.
// titleWeight is how many body matches one title match is worth. Two is the
// smallest value that lets a single domain term in the title outrank a body
// full of shared vocabulary, which is the failure that motivated it; higher
// would make the body decorative rather than secondary.
const titleWeight = 2

func relevanceOf(r Role, title, body string, paths []string, distinctive map[string]bool) int {
	score := r.ScopeOverlap(paths) * 2 // a declared path boundary is the stronger claim
	inTitle, inBody := taskWords(title), taskWords(body)
	for _, w := range declaredTerms(r) {
		if !distinctive[w] {
			continue
		}
		// Title and body are scored separately, never summed twice for the
		// same term: a term repeated in both is still one claim about domain.
		switch {
		case inTitle[w]:
			score += titleWeight
		case inBody[w]:
			score++
		}
	}
	return score
}

// declaredTerms is everything a role says about its own domain: the summary it
// was given plus the name it was given. Names are hyphenated by convention
// (`go-auditor`, `frontend-engineer`), so they split into exactly the terms
// that identify the specialty.
// Terms are singularized on both sides so the match is symmetric: a role
// declaring "compositions" and a task saying "composition" must meet, exactly
// as "seam"/"seams" must.
func declaredTerms(r Role) []string {
	raw := append(summaryWords(r.Summary), summaryWords(strings.ReplaceAll(r.Name, "-", " "))...)
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		out = append(out, singular(w))
	}
	return out
}

// distinctiveTerms returns the declared words claimed by exactly ONE candidate.
func distinctiveTerms(fit []Role) map[string]bool {
	count := map[string]int{}
	for _, r := range fit {
		// A term a role declares twice (in both its name and its summary) is
		// still ONE claimant — counting it twice would make it look shared and
		// silently drop the strongest signal a role can send.
		seen := map[string]bool{}
		for _, w := range declaredTerms(r) {
			if seen[w] {
				continue
			}
			seen[w] = true
			count[w]++
		}
	}
	out := map[string]bool{}
	for w, n := range count {
		if n == 1 {
			out[w] = true
		}
	}
	return out
}

// taskWords indexes the task's text by whole word, so a two-letter domain like
// "go" matches the language and not the "go" inside "going" or "algorithm".
// Both the singular and plural form are indexed, so a role declaring "seam"
// still matches a task that says "slice seams" — see singular.
func taskWords(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(strings.ToLower(text)) {
		w = strings.Trim(w, ".,:;!?\"'()-—/")
		if w == "" {
			continue
		}
		out[w] = true
		out[singular(w)] = true
	}
	return out
}

// singular strips one trailing plural "s", the single inflection that costs
// real matches: task titles name things in the plural ("slice seams", "the
// prompts") while a role declares the bare domain ("seam", "prompt"), and
// whole-word matching missed every one of those pairs — task 325 routed on
// price for exactly this reason.
//
// Deliberately not a stemmer. Two guards keep it from eating short technical
// terms, which is where a naive rule does damage: a word must be at least 4
// characters (so "js", "css" and "aws" survive), and a "ss" ending is never
// touched (so "access" and "css" are not truncated). Anything it gets wrong
// costs at most a missed match, never a wrong one, because the caller still
// requires the term to be DISTINCTIVE to one candidate.
func singular(w string) string {
	if len(w) < 4 || !strings.HasSuffix(w, "s") || strings.HasSuffix(w, "ss") {
		return w
	}
	return strings.TrimSuffix(w, "s")
}

// summaryWords reduces a role summary to candidate domain terms, dropping the
// filler every summary shares. The length floor is 2, not 3: "go", "ui" and
// "js" are exactly the kind of term that identifies a specialty.
func summaryWords(summary string) []string {
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "that": true,
		"an": true, "of": true, "to": true, "in": true, "on": true, "or": true,
		"its": true, "not": true, "never": true, "them": true, "their": true,
		"code": true, "work": true, "task": true, "tasks": true, "role": true,
		"best": true, "practices": true, "a": true, "is": true, "it": true,
	}
	seen := map[string]bool{}
	var out []string
	for _, w := range strings.Fields(strings.ToLower(summary)) {
		w = strings.Trim(w, ".,:;!?\"'()-—/")
		if len(w) < 2 || stop[w] || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}
