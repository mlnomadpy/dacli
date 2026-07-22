---
id: t-01KY5VX10Q350YDM6MP89Y2KWB
kind: task
created: 2026-07-22T21:32:26Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# I1: dacli ship --pr — push branches, open enriched PRs, merge via gh (PR-first integration)
## Acceptance
- [ ] dacli ship --pr (and integrate --pr) pushes each done task's branch, opens a PR via dacli pr (body from acceptance + findings + Fixes #issue; verify verdicts as review comments), and merges via gh pr merge — instead of a local git merge
- [ ] gated + operator-triggered; falls back to local merge with a warning when GitHub is unreachable; --no-merge opens the PRs and stops for human review
- [ ] committed by an agent; build + test green
## Log
