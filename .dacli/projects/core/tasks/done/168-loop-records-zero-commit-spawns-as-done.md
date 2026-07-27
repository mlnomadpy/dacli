---
id: t-01KYHWEWJM0H6JE1AVE38T2PH1
kind: task
created: 2026-07-27T13:33:04Z
created_by: a-root
owner: a-root
priority: must
---
# Loop records zero-commit spawns as done
## So that
a child that dies before committing is not recorded as completed work
## Acceptance
- [x] land status gates on rev-list --count trunk..branch greater than 0
- [x] zero-commit branch is treated as failed not merged
## Log
- 2026-07-27T23:03:03Z completed by a-root
