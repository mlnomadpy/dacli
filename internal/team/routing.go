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
	sort.SliceStable(fit, func(i, j int) bool {
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
