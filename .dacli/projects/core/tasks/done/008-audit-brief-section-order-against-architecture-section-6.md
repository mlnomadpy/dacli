---
id: t-01KY2K6YF6PQ8CNKG0WM8WGFBS
kind: task
created: 2026-07-21T15:02:51Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 1, probable: 2, pessimistic: 4}
github:
  issue: 13
  repo: mlnomadpy/dacli
---
# Audit brief section order against ARCHITECTURE section 6
## So that
the canonical brief example and the implementation cannot drift silently
## Acceptance
- [x] a finding reports match or mismatch of assembly order vs the doc, citing brief.go line numbers
## Log
- 2026-07-21T15:10:52Z finding by a-z4w30d506r: Brief order: 6 documented sections match §6, but impl adds 2 undocumented sections + drifted comment numbers
- 2026-07-21T15:12:00Z claimed by a-root
- 2026-07-21T15:12:00Z completed by a-root
