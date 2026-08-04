---
id: t-01KZ4WC8RQ43P83R69S5WYJNGE
kind: task
created: 2026-08-03T22:37:13Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Concurrent task add assigns duplicate seq numbers
## So that
two agents filing at once do not make both tasks unaddressable
## Acceptance
- [x] seq allocation is atomic or collision-safe
- [x] FindTask never reports ambiguous for concurrently created tasks
## Log
- 2026-08-04T00:06:45Z claimed by a-fixer-q3xzg4
- 2026-08-04T00:27:44Z accepted by a-root
- 2026-08-04T00:27:44Z verified by `go build ./...` (exit 0)
- 2026-08-04T00:27:44Z completed by a-root
