---
id: t-01KZ4WC8R19Y86V29E82QXEBXZ
kind: task
created: 2026-08-03T22:37:13Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# runs prune deletes live runs state
## So that
pruning cannot destroy a running agent's proc and transcript
## Acceptance
- [ ] prune skips runs whose process is still alive
## Log
