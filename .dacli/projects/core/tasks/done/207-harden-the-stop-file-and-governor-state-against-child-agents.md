---
id: t-01KZ4WC8QB4G7VEDM186DZ8AXM
kind: task
created: 2026-08-03T22:37:13Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Harden the STOP file and governor state against child agents
## So that
a child cannot disable the loop kill switch or reset its token budget
## Acceptance
- [x] the stop file is checked mid-wave not only between cycles
- [x] governor state is validated on read and a torn write cannot reset counters
## Log
- 2026-08-03T23:01:04Z completed by a-root
