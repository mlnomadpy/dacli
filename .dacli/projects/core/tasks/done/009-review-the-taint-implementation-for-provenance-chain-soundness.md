---
id: t-01KY38DMYQA60MVQGMMMK9VKM4
kind: task
created: 2026-07-21T21:13:30Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 1, probable: 2, pessimistic: 3}
---
# Review the taint implementation for provenance-chain soundness
## So that
the P4 blast-radius query can be trusted before it audits a real incident
## Acceptance
- [x] confirm origin survives event to note at sync, or find where it is lost
- [x] confirm taint is neither over-broad (clean projects) nor under-broad (misses derived notes)
- [x] flag any origin string that could evade the substring match
## Log
- 2026-07-21T21:33:16Z completed by a-root
