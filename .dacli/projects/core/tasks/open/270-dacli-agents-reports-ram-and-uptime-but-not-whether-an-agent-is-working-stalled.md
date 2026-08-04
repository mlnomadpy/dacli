---
id: t-01KZ6RTTBHRMG0VP8EVQ32V7JS
kind: task
created: 2026-08-04T16:13:44Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# dacli agents reports RAM and uptime but not whether an agent is working, stalled or stuck
## So that
an operator watching a wave sees which agent needs attention instead of a table of memory sizes
## Acceptance
- [ ] agents shows a per-agent state (thinking, acting, waiting, stalled, blocked, silent) using the same rules the dashboard already computes
- [ ] the state derivation is shared with the dashboard rather than reimplemented, or the duplication is deliberate and documented
- [ ] a stalled agent is visually distinct from a busy one without passing --tail
## Log
