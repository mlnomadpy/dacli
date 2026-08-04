---
id: t-01KZ53B8D240K4NWGWJQCXN4Q8
kind: task
created: 2026-08-04T00:39:00Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# checkLanded still reports a zero-commit branch as merged
## So that
pr status cannot call a dead spawn landed
## Acceptance
- [ ] vcs checkLanded has the same zero-commit guard as prLandStatus
## Log
- 2026-08-04T10:49:47Z claimed by a-maintainer-g2zmqk
