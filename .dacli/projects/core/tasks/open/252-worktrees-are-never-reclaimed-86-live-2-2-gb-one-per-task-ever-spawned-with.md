---
id: t-01KZ66036MVGVWKEQ796Q275YA
kind: task
created: 2026-08-04T10:44:34Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# Worktrees are never reclaimed: 86 live, 2.2 GB, one per task ever spawned with --worktree
## So that
a long-running project does not accumulate a gigabyte of dead checkouts per week
## Acceptance
- [ ] a worktree whose branch is merged or whose run is finished is pruned
- [ ] the prune is a command an operator can run and the loop calls it
## Log
