---
id: t-01KZ4W93AAMGPH7ETEFBNNAAMS
kind: task
created: 2026-08-03T22:35:29Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Fix remaining ghmirror audit findings: pagination truncation and unverified sync counts
## So that
a mature repo does not silently mirror only its first 1000 issues
## Acceptance
- [x] list calls detect a hit limit
- [x] synced counts only increment on verified writes
## Log
- 2026-08-04T00:29:24Z claimed by a-fixer-x41yjq
- 2026-08-04T11:32:44Z accepted by a-root
- 2026-08-04T11:32:44Z verified by `go test ./internal/features/ghmirror/` (exit 0)
- 2026-08-04T11:32:44Z completed by a-root
