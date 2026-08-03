---
id: t-01KYFN5MB1M1T4DGB4PRBJGV6T
kind: task
created: 2026-07-26T16:47:12Z
created_by: a-root
owner: a-root
priority: must
depends_on: [151]
github:
  issue: 248
  repo: mlnomadpy/dacli
---
# Refactor dashboard sections to shadcn-vue components (preserve every feature + the dark aesthetic)
## So that
the whole dashboard is consistent shadcn-vue, not a mix of hand-rolled and library components
## Acceptance
- [x] Overview, Task Board, Burndown, Burn Rate, Dependency Graph, and Agent Swarm are rebuilt on shadcn-vue primitives (Card/Table/Badge/Progress/Tooltip/etc.) + Tailwind; ALL existing behavior preserved (live swarm, burn alert threshold, DAG edges, agent-state badges) with no feature regression
- [x] Responsive + accessible; component tests updated and green; single-file build + go:embed still work; the mission-control dark look is preserved or improved
## Log
- 2026-07-26T17:23:17Z claimed by a-tja4fdtr3z
- 2026-07-26T17:41:26Z accepted by a-root
- 2026-07-26T17:41:26Z completed by a-root
- 2026-08-03T22:38:15Z a-tja4fdtr3z: PR opened: https://github.com/mlnomadpy/dacli/pull/111 (event 01KYFR7Q2THYHF439YGB5S12KF)
