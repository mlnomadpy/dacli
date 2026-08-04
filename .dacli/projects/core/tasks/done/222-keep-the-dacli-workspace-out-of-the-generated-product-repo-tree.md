---
id: t-01KZ4WDDHW19T5C480BM76PEM5
kind: task
created: 2026-08-03T22:37:51Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# Keep the dacli workspace out of the generated product repo tree
## So that
a generated app repo is not 80 percent agent bookkeeping files
## Acceptance
- [x] dacli new can gitignore the workspace while keeping the record branch
## Log
- 2026-08-04T10:50:16Z claimed by a-maintainer-zmqsrg
- 2026-08-04T11:32:59Z accepted by a-root
- 2026-08-04T11:32:59Z verified by `go test ./internal/features/wscore/ ./internal/features/ship/` (exit 0)
- 2026-08-04T11:32:59Z completed by a-root
