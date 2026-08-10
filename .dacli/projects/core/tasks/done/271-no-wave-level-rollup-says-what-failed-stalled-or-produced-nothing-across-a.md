---
id: t-01KZ6RV4BFE4HRHHDS5VTASG7C
kind: task
created: 2026-08-04T16:13:54Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# No wave-level rollup says what failed, stalled or produced nothing across a whole spawn wave
## So that
the operator reads one summary after a wave instead of opening six transcripts
## Acceptance
- [x] a command reports, per wave, which runs landed work, which produced nothing, which stalled and which were blocked
- [x] the rollup is derived from recorded run outcomes, not from re-parsing transcripts at read time
- [x] it names the recovery for each non-landing outcome rather than only labelling it
## Log
- 2026-08-09T22:58:36Z accepted by a-root
- 2026-08-09T22:58:36Z verified by `go build ./...` (exit 0)
- 2026-08-09T22:58:36Z deliverable: dacli/271-no-wave-level-rollup-says-what-failed-stalled-or-produced-nothing-across-a is merged into trunk
- 2026-08-09T22:58:36Z completed by a-root
