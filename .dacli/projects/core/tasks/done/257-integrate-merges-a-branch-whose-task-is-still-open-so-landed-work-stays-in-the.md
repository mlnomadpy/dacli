---
id: t-01KZ68VBDGT8RF99T7RJY2BAT6
kind: task
created: 2026-08-04T11:34:24Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# integrate merges a branch whose task is still open, so landed work stays in the backlog
## So that
next stops handing out work that shipped hours ago
## Acceptance
- [x] integrate refuses a named task that is not done, naming the task, unless --force
- [x] a regression test covers the not-done refusal and the --force override
- [x] the refusal message says which command closes the task
## Log
- 2026-08-04T12:03:06Z accepted by a-root
- 2026-08-04T12:03:06Z verified by `go test ./internal/features/vcs/` (exit 0)
- 2026-08-04T12:03:06Z completed by a-root
