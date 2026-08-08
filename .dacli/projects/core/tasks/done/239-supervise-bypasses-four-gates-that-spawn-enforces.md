---
id: t-01KZ53B8BRB68FAKQRGF7XYK9Z
kind: task
created: 2026-08-04T00:39:00Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# supervise bypasses four gates that spawn enforces
## So that
typing supervise instead of spawn cannot skip WIP, taint, budget and claim checks
## Acceptance
- [x] supervise enforces the same gates as spawn
- [x] a shared prologue makes a future gate apply to both
## Log
- 2026-08-04T09:48:52Z accepted by a-root
- 2026-08-04T09:48:52Z verified by `go test ./internal/features/execution/ -count=2 >/dev/null` (exit 0)
- 2026-08-04T09:48:52Z completed by a-root
