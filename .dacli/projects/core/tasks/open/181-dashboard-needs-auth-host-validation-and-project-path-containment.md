---
id: t-01KYHWFE4CANEFSB7JFVJCGENX
kind: task
created: 2026-07-27T13:33:22Z
created_by: a-root
owner: a-root
priority: should
---
# Dashboard needs auth Host validation and project path containment
## So that
the local dashboard cannot be read cross-workspace or via DNS rebinding
## Acceptance
- [ ] project query param is containment-checked like validRunID
- [ ] Host header is validated
## Log
