---
id: t-01KZPTRMP6NVQ7H3PCGB5VYWK4
kind: task
created: 2026-08-10T21:55:21Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# An unknown-flag refusal names what is wrong but never what is right
## Acceptance
- [x] a rejection for an unknown flag prints the command's Usage synopsis alongside the offending flag
- [x] the synopsis comes from Command.Usage, so the refusal and --help can never disagree
- [x] a test asserts the refusal contains both the bad flag name and the correct signature
## Log
- 2026-08-10T22:05:40Z accepted by a-root
- 2026-08-10T22:05:40Z verified by `go test ./internal/cli/ ./internal/clikit/ ./internal/store/` (exit 0)
- 2026-08-10T22:05:40Z deliverable: no dacli/348-an-unknown-flag-refusal-names-what-is-wrong-but-never-what-is-right branch — nothing to check against sprint/10
- 2026-08-10T22:05:40Z completed by a-root
