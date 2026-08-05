---
id: t-01KZ76GKZ3XM7JGXXGE1WF2D6V
kind: task
created: 2026-08-04T20:12:50Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# The github push task window scopes tasks but decisions ride along unscoped
## So that
a scoped push publishes exactly what was named, on a public repo where an unintended publish cannot be taken back
## Acceptance
- [ ] a task window scopes the decision and finding mirrors too, or the command refuses to combine a window with the unscoped mirrors
- [ ] the summary states how many issues of each kind will be created before any are, so the blast radius is visible
- [ ] a test asserts a windowed push creates no issue for an object outside the window
## Log
- 2026-08-04T20:31:40Z claimed by a-maintainer-qtd48g
