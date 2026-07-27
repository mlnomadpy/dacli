---
id: t-01KYHWFE4XXGTBD95Z0EGNPRHB
kind: task
created: 2026-07-27T13:33:22Z
created_by: a-root
owner: a-root
priority: could
---
# Loop should GC worktrees and merged branches after landing
## So that
the workspace does not accumulate gigabytes of stale state
## Acceptance
- [ ] landed tasks have their worktree and local branch removed
## Log
