---
id: t-01KZ6RVS0XJ9HRBF72VCYV549J
kind: task
created: 2026-08-04T16:14:16Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# accept moves the task file but leaves the move uncommitted, so doctor reports it in two folders
## So that
closing a task does not depend on the operator remembering a second step
## Acceptance
- [ ] accept stages its own record move, or doctor offers a fix that does
- [ ] a test covers close-then-inspect with no manual git step in between
## Log
