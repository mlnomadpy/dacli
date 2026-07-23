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
- [ ] doctor flags open/active tasks whose owner agent has no live process and is not root, recommending accept --force
- [ ] Covered by a test with an orphaned task fixture
## Log
