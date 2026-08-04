---
id: t-01KZ53B8EBPEMPKC4TMH21MJ4R
kind: task
created: 2026-08-04T00:39:00Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Integer flags parse four different ways and silently accept garbage
## So that
spawn --timeout 30s does not silently run with the default
## Acceptance
- [ ] clikit gains an Int helper and the discarded-error sites use it
## Log
