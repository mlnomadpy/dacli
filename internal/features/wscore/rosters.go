package wscore

import "github.com/mlnomadpy/dacli/internal/team"

// rosters are the starting role sets `dacli init --roster <name>` seeds, per
// docs/TEAM.md § 2. They are opinionated defaults, not universal truth — the
// docs are explicit that "editing it is expected". The point is to save the
// first-run typing, not to prescribe a team.
//
// Each role changes what an agent can DO (scope, grant, escalation), never
// just what it calls itself — the design rule in package team. A role that
// carves no boundary would be cosplay, so every seeded role either scopes a
// path or terminates an escalation chain.
var rosters = map[string][]team.Role{
	// software: the frontend / backend / sre / reviewer split TEAM.md § 2 names
	// for a web product. Implementers carve the tree by path; the reviewer sees
	// everything and demands the expensive model (cost routing, per the
	// decision note on reviewer×model).
	"software": {
		{Name: "frontend", Summary: "UI and client code", Kind: "implementer", Grant: "rw",
			Scope: []string{"web/**", "ui/**", "src/**"}, EscalateTo: []string{"reviewer", "human"}},
		{Name: "backend", Summary: "Services, data, and APIs", Kind: "implementer", Grant: "rw",
			Scope: []string{"internal/**", "cmd/**", "api/**", "server/**"}, EscalateTo: []string{"reviewer", "human"}},
		{Name: "sre", Summary: "Deploy, infra, and on-call", Kind: "implementer", Grant: "rw",
			Scope: []string{"deploy/**", "infra/**", "ops/**", ".github/**"}, EscalateTo: []string{"human"}},
		{Name: "reviewer", Summary: "Reviews changes across the tree", Kind: "reviewer", Grant: "ro",
			EscalateTo: []string{"human"}},
	},

	// research: the theorist / experimentalist / figure-editor split TEAM.md § 2
	// gives as the paper-repo example, scoped to the directories those roles
	// actually touch.
	"research": {
		{Name: "theorist", Summary: "Proofs and formal claims", Kind: "researcher", Grant: "rw",
			Skills: []string{"math-kernel-theory", "math-paper-audit"},
			Scope:  []string{"papers/**", "theory/**"}, EscalateTo: []string{"reviewer", "human"}},
		{Name: "experimentalist", Summary: "Experiments and results", Kind: "implementer", Grant: "rw",
			Skills: []string{"jax-ecosystem", "kaggle-cli-experiments"},
			Scope:  []string{"experiments/**", "scripts/**"}, EscalateTo: []string{"reviewer", "human"}},
		{Name: "figure-editor", Summary: "Figures and plots", Kind: "designer", Grant: "rw",
			Skills: []string{"tikz-figures"},
			Scope:  []string{"plots/**", "figures/**"}, EscalateTo: []string{"human"}},
		{Name: "reviewer", Summary: "Reviews claims and reproducibility", Kind: "reviewer", Grant: "ro",
			EscalateTo: []string{"human"}},
	},

	// solo: one generalist, no walls — the default for work that should not pay
	// for process (TEMPLATES.md § 2, and the empty-scope permissive rule in
	// package team). Seeded so `role list` is not empty on a solo workspace.
	"solo": {
		{Name: "maker", Summary: "Does the work, end to end", Kind: "implementer", Grant: "rw",
			EscalateTo: []string{"human"}},
	},
}
