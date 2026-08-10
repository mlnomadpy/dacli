---
id: t-01KZB4T2ZKQKPNT6CY1TXPZYDJ
kind: task
created: 2026-08-06T09:00:01Z
created_by: a-root
owner: a-root
priority: could
estimate: "{optimistic: 0.5, probable: 1, pessimistic: 2}"
---
# pr status reports DIRTY for behind-base branches, indistinguishable from a real content conflict
## So that
an operator triaging a stuck PR knows whether to merge main in or resolve conflicts
## Acceptance
- [x] the surfaced status distinguishes behind-base (mergeable after update) from content conflict, using the same probe the land phase uses
## Log
- 2026-08-08T12:13:06Z accepted by a-root
- 2026-08-08T12:13:06Z verified by `go build ./...` (exit 0)
- 2026-08-08T12:13:06Z completed by a-root
