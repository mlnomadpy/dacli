---
id: t-01KY9W5QWDRVDXGZFHV1QW2T3B
kind: task
created: 2026-07-24T10:54:09Z
created_by: a-root
owner: a-root
priority: must
---
# Dashboard: burn-rate + ceiling with a threshold that YELLS (research shortlist #1, RICE 4.8)
## So that
the one need all 4 segments share is met: catch overspend before it's a silent expensive failure
## Acceptance
- [x] A /api/burn endpoint exposes token/cost burn-rate over time from governor windowSpent + run actuals + calibrated role×model×runtime bands (data already recorded); handler test covers it
- [x] A Vue BurnRate component renders the rate and the ceiling, and CHANGES COLOR / alerts at ≥1.5× band (per docs/research: 'make the chart yell'), not just a passive line; wired into the dashboard; TS strict, ESLint clean, a component test
## Log
- 2026-07-24T10:54:25Z claimed by a-bx9nvegpnc
- 2026-07-24T11:10:31Z accepted by a-root
- 2026-07-24T11:10:31Z completed by a-root
