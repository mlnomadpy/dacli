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
- [ ] A regression test models a dead recorded leader whose numeric process group has been reused by an unrelated live process
- [ ] ReconcileRun does not report that run as alive and never signals the unrelated process group
- [ ] The fix preserves descendant-tree monitoring when the original leader is legitimately alive
- [ ] The relationship to task 285 and the residual limitation it left is recorded in the task notes or test name
- [ ] go test -race ./... passes
## Log
- Primary scope: `internal/procmon/**`, the liveness/reconciliation helpers in
  `internal/features/execution/**`, and their focused tests. Preserve PID-start
  identity and prove no unrelated process group is signaled.
- 2026-08-12T13:41:54Z claimed by a-codex-maintainer-4dj5bk
