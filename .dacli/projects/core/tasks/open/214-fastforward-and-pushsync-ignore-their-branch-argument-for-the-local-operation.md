---
id: t-01KZ4WCR5H1J88ECEEND2GWT3B
kind: task
created: 2026-08-03T22:37:29Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# FastForward and PushSync ignore their branch argument for the local operation
## So that
a loop started on a feature branch cannot fast-forward it onto trunk
## Acceptance
- [ ] the local operation verifies the checked-out branch matches
## Log
