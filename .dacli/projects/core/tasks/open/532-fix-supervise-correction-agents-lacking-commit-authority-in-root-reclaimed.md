---
id: t-01M13V42QDZE7CKYDFWYVB5YG5
kind: task
created: 2026-08-28T09:27:25Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 848
  repo: mlnomadpy/dacli
---
# Fix supervise correction agents lacking commit authority in root-reclaimed worktrees
## So that
bounded correction loops can finish verified work instead of preserving an uncommittable staged patch
## Acceptance
- [ ] A fixture creates a terminal child worktree, reclaims exact paths to root, and runs supervise on the same task without an inevitable commit-policy refusal.
- [ ] The supervised correction is either granted an audited task-scoped transfer it can commit under or materialized by root with preserved child attribution and verification evidence.
- [ ] Unrelated agents and paths outside the exact claim remain refused.
- [ ] A two-turn regression proves supervise does not spend every turn repeating the same ownership refusal.
- [ ] Mutation evidence and focused orchestration/VCS tests pass.
## Log
- 2026-08-28T09:28:52Z claimed by a-fixer-5xjt19
