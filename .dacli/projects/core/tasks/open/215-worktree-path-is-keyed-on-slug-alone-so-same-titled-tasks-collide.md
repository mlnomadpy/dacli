---
id: t-01KZ4WCR63P2K73JTS5633TGX0
kind: task
created: 2026-08-03T22:37:29Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# Worktree path is keyed on slug alone so same-titled tasks collide
## So that
two tasks with the same title do not share a worktree and commit to the wrong branch
## Acceptance
- [ ] worktree paths include the project and seq
## Log
