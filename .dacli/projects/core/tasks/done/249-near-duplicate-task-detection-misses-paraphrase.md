---
id: t-01KZ624RYTP0Y757EA90J943T8
kind: task
created: 2026-08-04T09:37:13Z
created_by: a-root
owner: a-root
priority: could
estimate: "{optimistic: 1, probable: 3, pessimistic: 8}"
---
# Near-duplicate task detection misses paraphrase
## So that
a reworded duplicate does not cost a whole agent run
## Acceptance
- [x] two tasks with the same meaning and no shared words are detected
- [x] any semantic backend is optional so the zero-dependency property holds
## Log
- 2026-08-04T11:36:13Z claimed by a-maintainer-kxp25t
- 2026-08-04T11:46:09Z accepted by a-root
- 2026-08-04T11:46:09Z verified by `go test ./internal/store/` (exit 0)
- 2026-08-04T11:46:09Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/307 (event 01KZ69HAZV5YSFDM7KQX051BHB)
