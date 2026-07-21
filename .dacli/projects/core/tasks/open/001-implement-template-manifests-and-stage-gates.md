---
id: t-01KY2J1BRB87WQHQMS9RVA5XCY
kind: task
created: 2026-07-21T14:42:19Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 4, probable: 8, pessimistic: 16}
---
# Implement template manifests and stage gates
## So that
controlled steps stop being spec fiction
## Acceptance
- [ ] gate predicates evaluate filled-not-present per TEMPLATES.md section 5
- [ ] solo template ships as default with zero gates
- [ ] stage advance refuses with the unmet list at exit 3
## Log
