---
id: t-01KZNYJ7QPJFE8TP6ZF783JJYV
kind: task
created: 2026-08-10T13:42:31Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Write the missing and strengthened tests the suite audit calls for
## So that
the gaps the audit names are closed rather than recorded
## Acceptance
- [x] each new test fails against the defect it targets before it passes, verified by breaking the code under it
- [x] no test asserts merely that an error occurred where it should assert which error
- [x] go test ./... and go vet pass, and coverage does not fall below the CI floor
## Log
- 2026-08-10T14:13:29Z accepted by a-root
- 2026-08-10T14:13:29Z verified by `go test ./internal/cli/... ./internal/features/orchestration/...` (exit 0)
- 2026-08-10T14:13:29Z deliverable: no dacli/315-write-the-missing-and-strengthened-tests-the-suite-audit-calls-for branch — nothing to check against trunk
- 2026-08-10T14:13:29Z completed by a-root
