---
id: t-01KZ4WC8SYR97DVKWZ1NW1MWJK
kind: task
created: 2026-08-03T22:37:13Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# resolveTrunkBranch returns HEAD on detached head and prefers a stale local main
## So that
the loop measures progress against the real trunk
## Acceptance
- [ ] detached head and unset origin HEAD resolve correctly or refuse
## Log
