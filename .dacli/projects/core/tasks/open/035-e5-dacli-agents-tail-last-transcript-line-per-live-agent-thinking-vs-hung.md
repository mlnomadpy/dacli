---
id: t-01KY53QHH3SQP4BEWPK55BZD7N
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: could
estimate: {optimistic: 1, probable: 2, pessimistic: 3}
---
# E5: dacli agents --tail — last transcript line per live agent (thinking vs hung)
## Acceptance
- [ ] dacli agents --tail shows each live agent's most recent transcript line beside its RAM/CPU, so a working agent is distinguishable from a hung one without manually tailing files
- [ ] committed on branch by an agent; build + test green
## Log
