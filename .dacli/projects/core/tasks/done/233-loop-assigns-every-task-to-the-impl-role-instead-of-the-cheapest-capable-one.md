---
id: t-01KZ51FTB69QAM2A7JR34AXG94
kind: task
created: 2026-08-04T00:06:32Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# loop assigns every task to the impl role instead of the cheapest capable one
## So that
the sprint spends cheap models on easy work automatically
## Acceptance
- [x] the loop picks the role via team.CheapestCapable using each task's Te
## Log
- 2026-08-04T00:06:46Z claimed by a-fixer-sphd68
- 2026-08-04T00:27:46Z accepted by a-root
- 2026-08-04T00:27:46Z verified by `go build ./...` (exit 0)
- 2026-08-04T00:27:46Z completed by a-root
