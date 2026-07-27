---
id: t-01KYHWEWMR9BFNRHABC6BXQTJD
kind: task
created: 2026-07-27T13:33:05Z
created_by: a-root
owner: a-root
priority: should
---
# loop max-cycles does not bound an idle loop
## So that
loop --max-cycles N terminates even on an empty backlog
## Acceptance
- [ ] the idle branch counts toward max-cycles
## Log
