---
id: t-01KZ6RW34DPA4NB11J0CSX6X23
kind: task
created: 2026-08-04T16:14:26Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 5, pessimistic: 10}"
github:
  issue: 342
  repo: mlnomadpy/dacli
---
# Agents in one wave are blind to each other, because a brief is frozen at spawn time
## So that
the fourth agent in a wave does not re-file what the second filed ten minutes ago
## Acceptance
- [ ] an agent can see findings and tasks its live siblings recorded after its own brief was assembled
- [ ] the mechanism does not require re-assembling a full brief mid-run
- [ ] a test covers two concurrent agents where the second sees the first's filing
## Log
- 2026-08-04T20:41:44Z claimed by a-maintainer-me4vk0
