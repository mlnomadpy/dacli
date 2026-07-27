---
id: t-01KYHWEB276HW6CJ2Y6W3TF95J
kind: task
created: 2026-07-27T13:32:47Z
created_by: a-root
owner: a-root
priority: must
---
# Reject path traversal in project slug and derived workspace paths
## So that
project add --slug cannot write or delete outside .dacli
## Acceptance
- [x] explicit --slug runs through Slugify or is rejected
- [x] workspace.dacli path assertion blocks .. escape
## Log
- 2026-07-27T23:03:03Z completed by a-root
