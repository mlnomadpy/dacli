---
id: t-01KZB4SAVS5A1TJ8020C1HVWC3
kind: task
created: 2026-08-06T08:59:36Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# spawn without --worktree checks the task branch out in the main tree and leaves no trunk
## So that
the first spawn in a fresh workspace cannot silently remove the trunk that ship needs to integrate into
## Acceptance
- [x] a no-worktree spawn onto a branch either refuses when it would switch the main checkout off trunk, or establishes trunk first and says so
- [x] ship with no trunk names the missing trunk and the spawn that caused it instead of reporting integrated-0 as success
## Log
- 2026-08-08T12:07:46Z accepted by a-root
- 2026-08-08T12:07:46Z verified by `go build ./...` (exit 0)
- 2026-08-08T12:07:46Z completed by a-root
