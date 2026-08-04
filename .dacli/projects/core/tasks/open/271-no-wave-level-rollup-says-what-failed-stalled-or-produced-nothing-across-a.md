---
id: t-01KZ6RV4BFE4HRHHDS5VTASG7C
kind: task
created: 2026-08-04T16:13:54Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
github:
  issue: 339
  repo: mlnomadpy/dacli
---
# No wave-level rollup says what failed, stalled or produced nothing across a whole spawn wave
## So that
the operator reads one summary after a wave instead of opening six transcripts
## Acceptance
- [ ] a command reports, per wave, which runs landed work, which produced nothing, which stalled and which were blocked
- [ ] the rollup is derived from recorded run outcomes, not from re-parsing transcripts at read time
- [ ] it names the recovery for each non-landing outcome rather than only labelling it
## Log
- 2026-08-04T20:41:54Z claimed by a-maintainer-qyec4d
