---
id: t-01KY59FND2699RMWSC7MAK1MNV
kind: task
created: 2026-07-22T16:10:34Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# F2: act on token bands — spawn-time budget from measured token cost (advise + enforce)
## Acceptance
- [ ] spawn --advise suggests a token budget from the role/model/runtime band's measured token-per-point (F1), not just wall-clock
- [ ] an optional per-run token ceiling warns/refuses when a band's expected cost exceeds it
- [ ] committed on branch by an agent; build + test green
## Log
