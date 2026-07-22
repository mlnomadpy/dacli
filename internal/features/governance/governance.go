// Package governance holds the honestly-stubbed command surface for the one
// subsystem still specified but not built: shortcut promotion. The subsystems
// this docstring once also listed — templates/gates (now stagegate), the
// GitHub mirror (ghmirror), skill compilation (skillforge), and verification
// panels (features/execution/verify.go) — have shipped. The remaining stub
// names its blocker and its spec: the roadmap, refusing to pretend.
package governance

import "github.com/mlnomadpy/dacli/internal/clikit"

var Commands = []clikit.Command{
	{Path: "shortcut promote", Brief: "Turn a repeated ad-hoc command into a shortcut", Run: clikit.Planned("ad-hoc command tracking — dacli only sees shortcut runs today, so there is nothing un-promoted to promote from", "docs/SHORTCUTS.md § promotion")},
}
