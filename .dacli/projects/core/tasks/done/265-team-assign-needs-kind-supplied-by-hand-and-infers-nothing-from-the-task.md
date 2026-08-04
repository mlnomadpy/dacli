---
id: t-01KZ6E2N0FCKCJBKQB9QWKA85N
kind: task
created: 2026-08-04T13:05:47Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# team assign needs --kind supplied by hand, and infers nothing from the task
## So that
the kind-aware half of routing is used by default instead of only when the caller already knows to ask for it
## Acceptance
- [x] assign infers a kind from the task (its own kind field, title verbs, or the phase) when --kind is absent
- [x] the inferred kind is printed so a wrong inference is visible rather than silent
## Log
- 2026-08-04T13:06:10Z claimed by a-maintainer-r31t6x
- 2026-08-04T14:35:54Z accepted by a-root
- 2026-08-04T14:35:54Z verified by `go test ./internal/features/teamops/` (exit 0)
- 2026-08-04T14:35:54Z completed by a-root
