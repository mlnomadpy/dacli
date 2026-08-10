---
id: t-01KZ6RW34DPA4NB11J0CSX6X23
kind: task
created: 2026-08-04T16:14:26Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 5, pessimistic: 10}"
---
# Agents in one wave are blind to each other, because a brief is frozen at spawn time
## So that
the fourth agent in a wave does not re-file what the second filed ten minutes ago
## Acceptance
- [x] an agent can see findings and tasks its live siblings recorded after its own brief was assembled
- [x] the mechanism does not require re-assembling a full brief mid-run
- [x] a test covers two concurrent agents where the second sees the first's filing
## Log
- 2026-08-09T22:58:37Z accepted by a-root
- 2026-08-09T22:58:37Z verified by `go build ./...` (exit 0)
- 2026-08-09T22:58:37Z deliverable: dacli/274-agents-in-one-wave-are-blind-to-each-other-because-a-brief-is-frozen-at-spawn is merged into trunk
- 2026-08-09T22:58:37Z completed by a-root
