---
id: t-01KZV16F64YVHKS00NQ9CZ5Q0C
kind: task
created: 2026-08-12T13:04:43Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 5, pessimistic: 8}"
github:
  issue: 462
  repo: mlnomadpy/dacli
---
# Resolve dead-leader process-group reuse false liveness
## So that
A recycled PGID cannot keep a completed dacli run alive or target an unrelated process tree
## Acceptance
- [x] A regression test models a dead recorded leader whose numeric process group has been reused by an unrelated live process
- [x] ReconcileRun does not report that run as alive and never signals the unrelated process group
- [x] The fix preserves descendant-tree monitoring when the original leader is legitimately alive
- [x] The relationship to task 285 and the residual limitation it left is recorded in the task notes or test name
- [x] go test -race ./... passes
## Log
- Primary scope: `internal/procmon/**`, the liveness/reconciliation helpers in
  `internal/features/execution/**`, and their focused tests. Preserve PID-start
  identity and prove no unrelated process group is signaled.
- 2026-08-12T13:41:54Z claimed by a-codex-maintainer-4dj5bk
- 2026-08-12T13:55:13Z completed by a-root
