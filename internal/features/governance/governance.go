// Package governance holds the honestly-stubbed command surface for
// subsystems that are specified but not built: templates/gates, the GitHub
// mirror, skill compilation, verification panels. Each stub names its
// blocker and its spec — the roadmap, refusing to pretend.
package governance

import "github.com/mlnomadpy/dacli/internal/clikit"

var Commands = []clikit.Command{
	{Path: "shortcut promote", Brief: "Turn a repeated ad-hoc command into a shortcut", Run: clikit.Planned("ad-hoc command tracking — dacli only sees shortcut runs today, so there is nothing un-promoted to promote from", "docs/SHORTCUTS.md § promotion")},
}
