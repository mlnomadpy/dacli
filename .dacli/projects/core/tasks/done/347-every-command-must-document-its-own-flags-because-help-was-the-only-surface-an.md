---
id: t-01KZPTR9BZ9AYSF7VBTQT4K0JM
kind: task
created: 2026-08-10T21:55:09Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Every command must document its own flags, because --help was the only surface an agent has
## Acceptance
- [x] every command whose handler rejects unknown flags declares a Usage synopsis, enumerated rather than sampled
- [x] a command taking no flags still prints its invocation, so --help always answers what shape the call takes
- [x] an invariant test fails when a flag-taking command is added without a synopsis, so the field cannot rot back
- [x] the note add signature from the report is printed in full by dacli note add --help
## Log
- 2026-08-10T21:55:39Z accepted by a-root
- 2026-08-10T21:55:39Z verified by `go test ./internal/cli/` (exit 0)
- 2026-08-10T21:55:39Z deliverable: no dacli/347-every-command-must-document-its-own-flags-because-help-was-the-only-surface-an branch — nothing to check against sprint/9
- 2026-08-10T21:55:39Z completed by a-root
