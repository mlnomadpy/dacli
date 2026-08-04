---
id: t-01KZ65ZG4XYFKTKT5YMH4HWNAG
kind: task
created: 2026-08-04T10:44:15Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 8}"
---
# Task seq is allocated against the working tree so two branches hand out the same number
## So that
a merged branch cannot produce two different tasks that share a reference
## Acceptance
- [x] filing a task on a branch cannot reuse a seq already taken on another unmerged branch
- [x] a regression test reproduces the collision across two branches
- [x] existing duplicate seqs are reconciled rather than silently left
## Log
- 2026-08-04T11:35:25Z claimed by a-maintainer-88hjw4
- 2026-08-04T11:50:25Z accepted by a-root
- 2026-08-04T11:50:25Z verified by `go test ./internal/store/ ./internal/features/insight/` (exit 0)
- 2026-08-04T11:50:25Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/308 (event 01KZ69S0D9DKJFDFF6MBKJA2D7)
