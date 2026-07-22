---
id: t-01KY55NX80QGRNMQNARZXAS7GQ
kind: task
created: 2026-07-22T15:04:04Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# E6: dacli ship — accept done tasks, integrate their branches, commit the record, push, in one operator command
## Acceptance
- [ ] dacli ship runs accept (verified) -> integrate --tasks -> a dacli-native workspace-record commit (never sweeping worktrees/runs/build) -> push, closing the manual wave tail the operator still does by hand
- [ ] each step is skippable/dry-runnable; a failure stops and reports, never half-ships
- [ ] committed on branch by an agent; build + test green
## Log
