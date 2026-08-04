---
id: t-01KZ52TWWZ5C9YXJ1533YWK88J
kind: task
created: 2026-08-04T00:30:04Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 6, pessimistic: 12}"
---
# Audit code quality across the codebase and file what is worth fixing
## So that
quality problems are recorded as evidence rather than opinion
## Acceptance
- [x] dead code, duplication and complexity hotspots are named with file and line
- [x] each finding states the concrete cost not a style preference
## Log
- 2026-08-04T09:35:57Z accepted by a-root
- 2026-08-04T09:35:57Z verified by `go test ./... >/dev/null` (exit 0)
- 2026-08-04T09:35:57Z completed by a-root
