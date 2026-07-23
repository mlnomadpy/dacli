---
id: t-01KY757X4E34DCKG201QQP6SHZ
kind: task
created: 2026-07-23T09:34:54Z
created_by: a-root
owner: a-root
priority: should
---
# dacli doctor detects orphaned tasks (owner is a finished agent) and suggests 'accept --force'
## Acceptance
- [x] doctor flags open/active tasks whose owner agent has no live process and is not root, recommending accept --force
- [x] Covered by a test with an orphaned task fixture
## Log
- 2026-07-23T11:58:40Z claimed by a-vrppnfvawm
- 2026-07-23T12:07:21Z accepted by a-root
- 2026-07-23T12:07:21Z completed by a-root
