---
id: t-01KZ702D3CV2TRGNWQD64JKT1C
kind: task
created: 2026-08-04T18:20:13Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 6}"
---
# An empty or lost DACLI_AGENT resolves to root with a read-write grant
## So that
losing an identity fails closed instead of silently escalating a child to the most privileged actor in the tree
## Acceptance
- [x] an empty DACLI_AGENT inside a spawned context is distinguished from no token at all
- [x] the distinction is testable without relying on the environment the test runs under
## Log
- 2026-08-06T08:07:18Z claimed by a-fixer-aa9apn
- 2026-08-08T11:07:20Z a-fixer-aa9apn: PR opened: https://github.com/mlnomadpy/dacli/pull/386 (event 01KZB2J3NPJH629KBZ8EX9GABR)
- 2026-08-08T11:54:42Z accepted by a-root
- 2026-08-08T11:54:42Z closed WITHOUT verification — no --verify command was given
- 2026-08-08T11:54:42Z completed by a-root
