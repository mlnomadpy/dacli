---
id: d-268-finalize-gone-but-running-detached-runs-lazily-on-observation-dacli-agents
kind: note
note_kind: decision
created: 2026-08-04T20:35:05Z
created_by: a-maintainer-1eed05
about: "[[268]]"
---
# 268: finalize gone-but-'running' detached runs lazily on observation (dacli agents sweeps), not via a daemon
## Chose
268: finalize gone-but-'running' detached runs lazily on observation (dacli agents sweeps), not via a daemon
## Rejected
finalizing in execRuntime's reaper goroutine after cmd.Process.Wait()
## Because
dacli is a stateless CLI with no daemon; the reaper goroutine only outlives the parent under 'dacli mcp serve', never under plain 'dacli spawn --detach' (the CLI exits immediately), and finalizing on leader-Wait is premature — forked group children may still be mid-commit (the dacli-177 concern runStillLive guards). A lazy sweep gated on runStillLive (leader AND group gone) in the canonical 'what is running' view (dacli agents) is safe, needs no cross-layer signature change, and is exactly what acceptance criterion 3 names ('reported by agents, not only by wait').
