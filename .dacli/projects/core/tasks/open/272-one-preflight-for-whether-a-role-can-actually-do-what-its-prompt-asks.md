---
id: t-01KZ6RVF7Z3CEW0FMAVV8MCB9F
kind: task
created: 2026-08-04T16:14:06Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
github:
  issue: 340
  repo: mlnomadpy/dacli
---
# One preflight for whether a role can actually do what its prompt asks
## So that
a capability mismatch is caught before a run burns rather than after it
## Acceptance
- [ ] a single command checks grant against runtime write capability, the dacli binary path against the allowlist, and the role prompt's named tools against what the runtime permits
- [ ] it reports every mismatch in one pass rather than failing on the first
- [ ] spawn runs it and refuses or warns per the existing convention for each class
## Log
- 2026-08-04T20:41:23Z claimed by a-maintainer-h7by24
