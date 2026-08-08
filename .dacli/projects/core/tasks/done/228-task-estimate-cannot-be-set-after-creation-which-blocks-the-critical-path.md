---
id: t-01KZ4WJ3BEP22AR6KBE4D63MS9
kind: task
created: 2026-08-03T22:40:24Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# task estimate cannot be set after creation which blocks the critical path
## So that
an existing backlog can be sized so critical-path and parallelism work
## Acceptance
- [x] an existing task's three-point estimate can be set from the CLI
- [x] critical-path returns a schedule once tasks are sized
## Log
- 2026-08-04T00:06:32Z completed by a-root
