---
id: t-01KZ53B8D240K4NWGWJQCXN4Q8
kind: task
created: 2026-08-04T00:39:00Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 8}"
---
# checkLanded still reports a zero-commit branch as merged
## So that
pr status cannot call a dead spawn landed
## Acceptance
- [x] vcs checkLanded has the same zero-commit guard as prLandStatus
## Log
- 2026-08-04T10:49:47Z claimed by a-maintainer-g2zmqk
- 2026-08-04T11:33:04Z accepted by a-root
- 2026-08-04T11:33:04Z verified by `go test ./internal/features/vcs/` (exit 0)
- 2026-08-04T11:33:04Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/295 (event 01KZ67C1E37AG7Y24KDD2CD1DY)
