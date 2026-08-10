---
id: t-01KZB4RXM4B97VT8VQJ8KF3MPE
kind: task
created: 2026-08-06T08:59:23Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# loop never closes tasks in a no-remote workspace, so next re-picks them forever
## So that
a repo with no GitHub remote can still finish work: local merge into trunk counts as landing, and accept runs against it
## Acceptance
- [x] with no remote and no github link, a loop cycle that merges a task branch into trunk closes the task (completed-by stamped) instead of deferring accept forever
- [x] the land phase says which confirmation path it used (PR merge vs local trunk merge) rather than printing the PR message unconditionally
- [x] a test runs a cycle in a workspace with no origin and asserts the task record moves to done
## Log
- 2026-08-08T12:02:06Z accepted by a-root
- 2026-08-08T12:02:06Z verified by `go test ./internal/features/orchestration/...` (exit 0)
- 2026-08-08T12:02:06Z completed by a-root
