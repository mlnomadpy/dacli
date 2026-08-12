---
id: t-01KZV4SAVNZ1JXMJ61RRQCJYPY
kind: task
created: 2026-08-12T14:07:27Z
created_by: a-codex-loop-auditor-yq4y7k
owner: a-root
priority: must
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: [372]
github:
  issue: 468
  repo: mlnomadpy/dacli
---
# Fix run reconciliation to preserve authenticated descendants after leader exit
## Acceptance
- [x] A Unix regression fixture starts a recorded group leader that forks a helper, lets the leader exit, and proves dacli wait/reconciliation still reports the real helper live until it exits
- [x] The same focused tests prove a dead leader plus an unrelated process group that reused the numeric PGID remains dead and kill never signals that unrelated group
- [x] proc.txt or an equivalent durable run record carries enough process identity to distinguish genuine surviving descendants from a recycled group after the original leader exits
- [x] The task 177 and task 369 invariants are both named in the regression tests, and go test -race ./... passes
## Log
- 2026-08-12T14:10:21Z adopted by a-root (owner a-codex-loop-auditor-yq4y7k orphaned)
- 2026-08-12T14:11:28Z claimed by a-codex-maintainer-1ecns6
- 2026-08-12T15:25:28Z completed by a-root
