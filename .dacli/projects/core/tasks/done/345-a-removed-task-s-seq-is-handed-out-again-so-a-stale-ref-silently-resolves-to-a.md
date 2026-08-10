---
id: t-01KZPRKM1NS8NR1QDV1DXRV6QE
kind: task
created: 2026-08-10T21:17:39Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# A removed task's seq is handed out again, so a stale ref silently resolves to a different task
## Acceptance
- [x] a seq freed by task rm is not reused while any agent could still hold the old ref
- [x] the reuse hazard is closed even when .dacli is gitignored, where gitTaskSeqCeiling only sees what has been shipped to the record branch
- [x] a test creates a task, removes it, creates another, and asserts the second does not take the first's seq
## Log
- 2026-08-10T21:45:15Z accepted by a-root
- 2026-08-10T21:45:15Z verified by `go test ./internal/store/` (exit 0)
- 2026-08-10T21:45:15Z deliverable: no dacli/345-a-removed-task-s-seq-is-handed-out-again-so-a-stale-ref-silently-resolves-to-a branch — nothing to check against sprint/7
- 2026-08-10T21:45:15Z completed by a-root
