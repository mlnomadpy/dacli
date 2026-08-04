---
id: t-01KZ4WCR4VHYZX267PM5042H9R
kind: task
created: 2026-08-03T22:37:29Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# gitx deadlines do not actually bound a git child
## So that
a credential prompt or wedged grandchild cannot hang the CLI
## Acceptance
- [ ] runWithTimeout sets WaitDelay like execution does
## Log
