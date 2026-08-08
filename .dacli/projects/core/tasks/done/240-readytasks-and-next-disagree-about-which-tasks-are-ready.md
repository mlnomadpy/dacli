---
id: t-01KZ53B8CD9B17WPMK3F5BNZMV
kind: task
created: 2026-08-04T00:39:00Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# readyTasks and next disagree about which tasks are ready
## So that
next does not recommend work the loop will never pick
## Acceptance
- [x] one predicate decides readiness for both
## Log
- 2026-08-04T09:50:28Z accepted by a-root
- 2026-08-04T09:50:28Z verified by `go test ./... >/dev/null` (exit 0)
- 2026-08-04T09:50:28Z completed by a-root
