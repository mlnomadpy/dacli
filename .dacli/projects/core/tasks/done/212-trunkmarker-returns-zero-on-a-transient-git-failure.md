---
id: t-01KZ4WC8TK789RGNJ772FJQ2F9
kind: task
created: 2026-08-03T22:37:13Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# trunkMarker returns zero on a transient git failure
## So that
a wedged git does not fake a stall then a burst of progress
## Acceptance
- [x] a failed measurement is distinguishable from zero progress
## Log
- 2026-08-03T23:01:04Z completed by a-root
