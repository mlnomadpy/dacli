---
id: t-01KZ4WC8RQ43P83R69S5WYJNGE
kind: task
created: 2026-08-03T22:37:13Z
created_by: a-root
owner: a-root
priority: must
---
# Concurrent task add assigns duplicate seq numbers
## So that
two agents filing at once do not make both tasks unaddressable
## Acceptance
- [ ] seq allocation is atomic or collision-safe
- [ ] FindTask never reports ambiguous for concurrently created tasks
## Log
