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
- [x] the preamble a ro child receives describes the propose-and-sync path, not commit and pr
- [x] a test asserts the ro and rw preambles differ on exactly the actions the grant governs
## Log
- 2026-08-06T08:07:17Z claimed by a-junior-fqc361
- 2026-08-06T08:25:16Z accepted by a-root
- 2026-08-06T08:25:16Z closed WITHOUT verification — no --verify command was given
- 2026-08-06T08:25:16Z completed by a-root
- 2026-08-08T11:07:20Z a-junior-fqc361: PR opened: https://github.com/mlnomadpy/dacli/pull/385 (event 01KZB240MCCMKC1CZPE4CPKDSN)
