---
id: t-01KZ4WC8SB2VC3H4M8GSDAJXMT
kind: task
created: 2026-08-03T22:37:13Z
created_by: a-root
owner: a-root
priority: should
---
# Acceptance proposals are consumed before the close is durable
## So that
a failed CloseTask cannot make completed work permanently invisible
## Acceptance
- [ ] proposals are marked applied only after the task close succeeds
## Log
