---
id: t-01KZP1MKWP0CSCYRTN5BKHE1XN
kind: task
created: 2026-08-10T14:36:15Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# team assign infers kind from any word in the title, so a task that mentions an audit is routed to a reviewer
## So that
routing reflects what the task asks an agent to DO, not a noun that happens to appear in its title
## Acceptance
- [x] the kind is inferred from the leading verb rather than the first keyword found anywhere in the title, so "Write the tests the suite audit calls for" routes to an implementer
- [x] a role whose charter forbids implementing is never returned for implementer work, whatever the cost ranking says
- [x] a test drives two titles differing only by an incidental noun and asserts they route to the same kind
## Log
- 2026-08-10T14:47:14Z accepted by a-root
- 2026-08-10T14:47:14Z verified by `go test ./internal/features/teamops/...` (exit 0)
- 2026-08-10T14:47:14Z deliverable: no dacli/318-team-assign-infers-kind-from-any-word-in-the-title-so-a-task-that-mentions-an branch — nothing to check against trunk
- 2026-08-10T14:47:14Z completed by a-root
