---
id: t-01KYFN5MB1M1T4DGB4PRBJGV6T
kind: task
created: 2026-07-26T16:47:12Z
created_by: a-root
owner: a-root
priority: must
depends_on: [151]
---
# Refactor dashboard sections to shadcn-vue components (preserve every feature + the dark aesthetic)
## So that
the whole dashboard is consistent shadcn-vue, not a mix of hand-rolled and library components
## Acceptance
- [ ] Overview, Task Board, Burndown, Burn Rate, Dependency Graph, and Agent Swarm are rebuilt on shadcn-vue primitives (Card/Table/Badge/Progress/Tooltip/etc.) + Tailwind; ALL existing behavior preserved (live swarm, burn alert threshold, DAG edges, agent-state badges) with no feature regression
- [ ] Responsive + accessible; component tests updated and green; single-file build + go:embed still work; the mission-control dark look is preserved or improved
## Log
