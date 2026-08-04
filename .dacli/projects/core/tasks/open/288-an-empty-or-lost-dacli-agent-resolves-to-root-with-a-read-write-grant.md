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
- [ ] an empty DACLI_AGENT inside a spawned context is distinguished from no token at all
- [ ] the distinction is testable without relying on the environment the test runs under
## Log
