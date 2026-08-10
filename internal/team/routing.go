package team

import (
	"sort"
	"strings"
)

// Model cost tiers, cheapest first. These are ORDERING ranks, not prices: the
// routing question is only ever "which of these is cheaper", and a rank
// survives price changes that a hardcoded dollar figure would not.
//
// The list is deliberately short and matched by substring, because real configs
// carry vendor-prefixed, dated names (`claude-3-5-sonnet-20241022`). A model
// this build has never heard of ranks as EXPENSIVE, not cheap — routing real
// work to something nobody priced is the failure mode worth designing against,
// and an unset model is unpriced rather than free (dacli 231).
var modelTiers = []struct {
	match string
	tier  int
}{
	{"haiku", 1},
	{"sonnet", 2},
	{"opus", 3},
}

// tierUnknown sorts after every known model.
const tierUnknown = 99

// ModelTier ranks a model by cost, cheapest lowest. Unknown or unset models
// rank last so they are never preferred by accident.
func ModelTier(model string) int {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return tierUnknown
	}
	for _, t := range modelTiers {
		if strings.Contains(m, t.match) {
			return t.tier
		}
	}
	return tierUnknown
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
func CheapestCapableFor(roles []Role, kind string, te float64, files []string, taskText string) (Role, bool) {
	var fit []Role
	for _, r := range roles {
		if !strings.EqualFold(r.Kind, kind) {
			continue
		}
		if r.MaxPoints > 0 && te > r.MaxPoints {
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
		rel[r.Name] = relevanceOf(r, taskText, files, distinctive)
		if rel[r.Name] > 0 {
			discriminates = true
		}
	}

	sort.SliceStable(fit, func(i, j int) bool {
		if discriminates {
			if ri, rj := rel[fit[i].Name], rel[fit[j].Name]; ri != rj {
				return ri > rj
			}
		}
		ti, tj := ModelTier(fit[i].Model), ModelTier(fit[j].Model)
		if ti != tj {
			return ti < tj
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
	if r.MaxPoints <= 0 {
		return 1 << 30
	}
	return r.MaxPoints
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
// Two signals, because one is often unavailable. Scope globs answer when the
// task names files; when it names none — as every audit task filed this
// session did — the role's SUMMARY is the only declaration of what it does.
func relevanceOf(r Role, taskText string, paths []string, distinctive map[string]bool) int {
	score := r.ScopeOverlap(paths) * 2 // a declared path boundary is the stronger claim
	words := taskWords(taskText)
	for _, w := range summaryWords(r.Summary) {
		if distinctive[w] && words[w] {
			score++
		}
	}
	return score
}

// distinctiveTerms returns the summary words claimed by exactly ONE candidate.
func distinctiveTerms(fit []Role) map[string]bool {
	count := map[string]int{}
	for _, r := range fit {
		for _, w := range summaryWords(r.Summary) {
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
func taskWords(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(strings.ToLower(text)) {
		out[strings.Trim(w, ".,:;!?\"'()-—/")] = true
	}
	return out
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
