---
id: t-01KY757X3WZKWBB6NS89QAY8A6
kind: task
created: 2026-07-23T09:34:54Z
created_by: a-root
owner: a-root
priority: should
---
# Fix misleading '(read-only grant)' messages: the real reason is non-ownership, not grant
## Acceptance
- [ ] The 4 task-mutation messages distinguish 'not the owner (propose an event, or accept --force as root)' from an actual ro-grant refusal
- [ ] Wording verified; no behavior change beyond the message
## Log
