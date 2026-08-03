---
id: t-01KZ4E0YP7F0Y9ZXJYT48HCWHQ
kind: task
created: 2026-08-03T18:26:22Z
created_by: a-root
owner: a-root
priority: should
---
# Replace hardcoded Go roles and gofmt with stack detection
## So that
a python or typescript app is not driven by go-auditor and gofmt
## Acceptance
- [ ] impl and review roles derive from the detected stack
- [ ] format and test commands are per-project
## Log
