// Package governance holds the honestly-stubbed command surface for
// subsystems that are specified but not built: templates/gates, the GitHub
// mirror, skill compilation, verification panels. Each stub names its
// blocker and its spec — the roadmap, refusing to pretend.
package governance

import "github.com/mlnomadpy/dacli/internal/clikit"

var Commands = []clikit.Command{
	{Path: "github doctor", Brief: "Probe gh, auth, repo access, and Projects scope", Run: clikit.Planned("the issue/project mirror", "docs/GITHUB.md")},
	{Path: "github link", Brief: "Bind a project to a repo and a GitHub Project", Run: clikit.Planned("the issue/project mirror", "docs/GITHUB.md")},
	{Path: "github sync", Brief: "Sync with GitHub Issues and Projects (--dry-run)", Run: clikit.Planned("the issue/project mirror with marker-based idempotency", "docs/GITHUB.md § 4")},
	{Path: "github pull", Brief: "Inbound only: fetch remote changes as events", Run: clikit.Planned("inbound humans-as-events", "docs/GITHUB.md § 3")},
	{Path: "github push", Brief: "Outbound only: mirror local structure", Run: clikit.Planned("the issue/project mirror", "docs/GITHUB.md")},

	{Path: "skill add", Brief: "Author a workspace skill", Run: clikit.Planned("skill compilation", "docs/SKILLS.md")},
	{Path: "skill list", Brief: "Workspace skills with sizes and delivery floors", Run: clikit.Planned("skill compilation", "docs/SKILLS.md")},
	{Path: "skill show", Brief: "One skill: body, resources, est. tokens", Run: clikit.Planned("skill compilation", "docs/SKILLS.md")},
	{Path: "skill import", Brief: "Ingest a native skill tree losslessly", Run: clikit.Planned("skill compilation", "docs/SKILLS.md")},
	{Path: "skill compile", Brief: "Materialize skills for a role on a runtime (--dry-run)", Run: clikit.Planned("the fidelity ladder (native/context/inline)", "docs/SKILLS.md § 3")},
	{Path: "skill promote", Brief: "Owner-gated promotion of a lesson into a skill", Run: clikit.Planned("lessons (PROPOSALS P1) landing first — nothing to promote yet", "docs/SKILLS.md § 6")},

	{Path: "shortcut promote", Brief: "Turn a repeated ad-hoc command into a shortcut", Run: clikit.Planned("ad-hoc command tracking — dacli only sees shortcut runs today, so there is nothing un-promoted to promote from", "docs/SHORTCUTS.md § promotion")},
}
