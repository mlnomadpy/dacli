---
id: t-01KZ7021YCS3DYPBTRETZ96BZB
kind: task
created: 2026-08-04T18:20:01Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# The protocol preamble tells a read-only agent to commit and open a PR, which it cannot do
## So that
an agent is not instructed to attempt what its own grant forbids, and does not read the refusal as its own mistake
## Acceptance
- [ ] the preamble a ro child receives describes the propose-and-sync path, not commit and pr
- [ ] a test asserts the ro and rw preambles differ on exactly the actions the grant governs
## Log
- 2026-08-06T08:07:17Z claimed by a-junior-fqc361
