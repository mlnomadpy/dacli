---
id: t-01KZ4X918T0MESRD43CMF5V1QY
kind: task
created: 2026-08-03T22:52:55Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1, pessimistic: 3}"
---
# spawn --advise and loop --advise mean opposite things
## So that
a flag named advise cannot cost money when the operator expected a preview
## Acceptance
- [x] advise previews without acting on both commands, or the spawn flag is renamed
## Log
- 2026-08-04T11:36:07Z claimed by a-maintainer-vrmdqz
- 2026-08-04T11:51:05Z accepted by a-root
- 2026-08-04T11:51:05Z verified by `go test ./internal/features/execution/ ./internal/prompts/` (exit 0)
- 2026-08-04T11:51:05Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/311 (event 01KZ69TH14Q3WXM2ZRXT299PQF)
