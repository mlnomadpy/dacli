---
id: t-01KZ6A3CDSN23JKVHX0VV3XG62
kind: task
created: 2026-08-04T11:56:16Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# accept and task writes from inside a worktree land in the main workspace, not the branch
## So that
an agent's record of its own work travels with its branch instead of racing every sibling in one shared store
## Acceptance
- [ ] the resolved workspace root for a command run inside a worktree is documented and deliberate
- [ ] if main-workspace resolution is intended, spawn tells the agent so; if not, the worktree's own .dacli is used
## Log
- 2026-08-04T12:35:52Z claimed by a-maintainer-xckjk8
