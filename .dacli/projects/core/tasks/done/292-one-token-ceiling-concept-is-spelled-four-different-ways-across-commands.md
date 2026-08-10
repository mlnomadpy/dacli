---
id: t-01KZ70400DTHKRV0HAT29D283D
kind: task
created: 2026-08-04T18:21:05Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 3, pessimistic: 6}"
---
# One token-ceiling concept is spelled four different ways across commands
## So that
an agent can predict the flag for a command it has not used from the ones it has
## Acceptance
- [x] the ceiling has one name, with the others accepted as documented aliases or removed
- [x] any remaining homonym is called out in help so a wrong guess fails loudly rather than silently meaning something else
## Log
- 2026-08-09T23:08:07Z accepted by a-root
- 2026-08-09T23:08:07Z verified by `go test ./internal/clikit/...` (exit 0)
- 2026-08-09T23:08:07Z deliverable: dacli/292-one-token-ceiling-concept-is-spelled-four-different-ways-across-commands is merged into trunk
- 2026-08-09T23:08:07Z completed by a-root
