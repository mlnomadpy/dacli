---
id: t-01KZPG7MNVWCZQQ11ZM3KBQJKC
kind: task
created: 2026-08-10T18:51:18Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# There is no way to reopen a wrongly-closed task, or to remove one that should never have existed
## Acceptance
- [x] a closed task can be reopened through a command, with the reopen and its reason recorded in the task log
- [x] a task can be removed, refusing if anything references it, so a probe or duplicate does not have to be deleted by hand
- [x] neither command silently unchecks acceptance boxes: the boxes it clears and why are stated in the output
## Log
- 2026-08-10T19:22:12Z accepted by a-root
- 2026-08-10T19:22:12Z verified by `go test ./internal/features/planning/ ./internal/cli/ ./internal/store/` (exit 0)
- 2026-08-10T19:22:12Z deliverable: no dacli/340-there-is-no-way-to-reopen-a-wrongly-closed-task-or-to-remove-one-that-should branch — nothing to check against trunk
- 2026-08-10T19:22:12Z completed by a-root
