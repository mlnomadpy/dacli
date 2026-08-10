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
- [x] agents shows a per-agent state (thinking, acting, waiting, stalled, blocked, silent) using the same rules the dashboard already computes
- [x] the state derivation is shared with the dashboard rather than reimplemented, or the duplication is deliberate and documented
- [x] a stalled agent is visually distinct from a busy one without passing --tail
## Log
- 2026-08-05T13:35:31Z claimed by a-fixer-015xkz
- 2026-08-05T14:01:06Z accepted by a-root
- 2026-08-05T14:01:06Z closed WITHOUT verification — no --verify command was given
- 2026-08-05T14:01:06Z completed by a-root
- 2026-08-08T11:07:20Z a-fixer-015xkz: PR opened: https://github.com/mlnomadpy/dacli/pull/375 (event 01KZ935N07HQM345038BYX2KPA)
