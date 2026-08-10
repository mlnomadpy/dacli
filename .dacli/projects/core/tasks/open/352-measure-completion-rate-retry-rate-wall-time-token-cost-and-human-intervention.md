---
id: t-01KZPWSHXP2Y1TX9VDPVV01665
kind: task
created: 2026-08-10T22:30:48Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Measure completion rate, retry rate, wall time, token cost and human-intervention rate across repeatable scenarios
## Acceptance
- [ ] a command reports these five figures for a named window, derived from the run records and event log already on disk
- [ ] the figures are defined in the output, so a reader knows what counts as a retry and what counts as an intervention
- [ ] a scenario can be re-run and its numbers compared, which is what makes them a measurement rather than a snapshot
## Log
