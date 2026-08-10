---
id: t-01KZP3B02AN20SRRRVVCZEY4E9
kind: task
created: 2026-08-10T15:05:57Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# the review phase silently never runs when its role is capped, because the anchor is never sized
## So that
an idle loop that generates no work says why, instead of reporting a healthy empty backlog forever
## Acceptance
- [x] a failed review spawn is reported with its refusal text, never discarded
- [x] the standing anchor carries an estimate so a capacity-capped review role can accept it
- [x] a test asserts the review phase surfaces a spawn refusal rather than swallowing it
## Log
- 2026-08-10T15:08:34Z claimed by a-junior-r5smp7
- 2026-08-10T15:14:48Z a-junior-r5smp7: PR opened: https://github.com/mlnomadpy/dacli/pull/414 (event 01KZP3T0KGTCDHNVWMGGB32GSW)
- 2026-08-10T16:13:14Z accepted by a-root (applied 1 proposal(s))
- 2026-08-10T16:13:14Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T16:13:14Z deliverable: dacli/320-the-review-phase-silently-never-runs-when-its-role-is-capped-because-the-anchor is merged into trunk
- 2026-08-10T16:13:14Z completed by a-root
