---
id: t-01KYHWEWQ6W5H8632RA7Q5FZ8M
kind: task
created: 2026-07-27T13:33:05Z
created_by: a-root
owner: a-root
priority: should
---
# dacli wait checks only the leader PID not the process group
## So that
the loop does not proceed to land while children are mid-commit
## Acceptance
- [ ] wait uses GroupAlive
## Log
