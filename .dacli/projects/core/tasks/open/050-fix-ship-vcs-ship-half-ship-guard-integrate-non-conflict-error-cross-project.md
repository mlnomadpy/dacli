---
id: t-01KY59YW0ACC3C3GMWGW0BS8EY
kind: task
created: 2026-07-22T16:18:52Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# FIX ship/vcs: ship half-ship guard, integrate non-conflict error, cross-project refs, message
## Acceptance
- [ ] ship never half-ships: if integrate's clean-tree guard no-ops because accept dirtied .dacli, ship detects it and does not commit+push a partial record; the record commit reports only branches ACTUALLY merged
- [ ] cmdIntegrate propagates a real non-conflict merge failure instead of mislabelling it a conflict and swallowing to exit 0 (regression of the 018 fix)
- [ ] ship resolves --tasks refs unambiguously across a multi-project done list (qualify by project, not bare seq)
- [ ] committed by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T16:19:09Z claimed by a-jjwx3z556n
