---
id: t-01KZ4WDDHBY84GE1AP0EJMS1H7
kind: task
created: 2026-08-03T22:37:51Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# gh calls never pass --repo so writes can target the wrong repository
## So that
issues land in the repo the project is linked to
## Acceptance
- [x] gh writes target the stored repo explicitly
## Log
- 2026-08-04T11:57:24Z accepted by a-root
- 2026-08-04T11:57:24Z verified by `go test ./internal/features/ghmirror/` (exit 0)
- 2026-08-04T11:57:24Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/315 (event 01KZ6A5W9GZVPEBJTAXZRWZ622)
