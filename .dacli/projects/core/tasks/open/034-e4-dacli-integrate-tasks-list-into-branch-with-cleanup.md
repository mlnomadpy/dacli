---
id: t-01KY53QHGRNRT2NQ7BYAYJ9BWS
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 4, pessimistic: 6}
---
# E4: dacli integrate --tasks <list> --into <branch> with cleanup
## Acceptance
- [ ] dacli integrate takes an explicit task list and a target branch (not just into=main), merges each done-task branch, reports per-task merged/conflict, and removes the worktree + branch on success
- [ ] a conflict blocks that one task and continues the rest (or stops, documented), never half-merges
- [ ] committed on branch by an agent; build + test green
## Log
