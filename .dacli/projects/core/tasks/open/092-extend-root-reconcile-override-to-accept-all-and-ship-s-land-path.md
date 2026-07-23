---
id: t-01KY757X2TJ3CBMT83ZKPBGF35
kind: task
created: 2026-07-23T09:34:54Z
created_by: a-root
owner: a-root
priority: must
---
# Extend root reconcile override to 'accept --all' and ship's land path
## Acceptance
- [ ] accept --all --force (root) reconciles tasks owned by finished agents; ship passes it so the loop auto-closes orphaned agent tasks
- [ ] A test covers acceptAll closing a task owned by another agent under --force
## Log
