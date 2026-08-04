---
id: t-01KZ53GAZN51WVDC7Y927H8G6K
kind: task
created: 2026-08-04T00:41:46Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# ghmirror re-reads every note per task, 238x waste
## So that
github push does not burn 0.58s and 341MB of garbage on local file IO
## Acceptance
- [ ] ListNotes is hoisted above the per-task loop
## Log
