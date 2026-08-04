---
id: t-01KZ53GB099M2Q5YAQQ7JKG4TT
kind: task
created: 2026-08-04T00:41:46Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# brief Assemble walks the event log three times and the task tree twice
## So that
dacli context is not 37 percent redundant file reads
## Acceptance
- [x] one event walk filtered in memory, and the task tree read once
## Log
- 2026-08-04T09:46:32Z accepted by a-root
- 2026-08-04T09:46:32Z verified by `go test ./... >/dev/null` (exit 0)
- 2026-08-04T09:46:32Z completed by a-root
