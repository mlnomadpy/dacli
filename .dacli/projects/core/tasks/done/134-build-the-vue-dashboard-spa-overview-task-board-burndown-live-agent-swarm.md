---
id: t-01KY8KW3XN37VGXCEVQBTDE6R9
kind: task
created: 2026-07-23T23:09:51Z
created_by: a-root
owner: a-root
priority: must
depends_on: [133]
---
# Build the Vue dashboard SPA: overview, task board/burndown, live agent swarm
## So that
the dashboard is a polished, live mission-control UI, not a static table
## Acceptance
- [x] Components (per DESIGN.md) consume the API via a Pinia store; the live agent swarm auto-updates (polling or SSE) so a running loop's agents appear live; overview + task-status/burndown render; loading/empty/error states handled
- [x] TypeScript strict, ESLint clean, responsive + accessible per DESIGN; Vitest tests cover the store and a key component
## Log
- 2026-07-24T01:01:20Z claimed by a-j846nahs42
- 2026-07-24T01:18:51Z accepted by a-root
- 2026-07-24T01:18:51Z completed by a-root
