---
id: t-01KZP8HANTQJW7EM8DBRS808ZC
kind: task
created: 2026-08-10T16:36:47Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Trace one user-invoked verb end to end across slice seams and name where the report diverges from the effect
## Acceptance
- [x] one verb (ship, loop, accept, spawn, integrate, or sync) traced step by step across every slice boundary it crosses
- [x] each handoff records what the consumer assumes the producer did, and the code path where that assumption fails
- [x] the half-failure state of each step is named, and whether the next run can recover from it unaided
- [x] any finding is stated as a concrete step sequence producing a wrong state or a wrong report, not a layering opinion
## Log
- 2026-08-10T16:37:31Z claimed by a-seam-auditor-qzz7rf
- 2026-08-10T16:42:06Z finding by a-seam-auditor-qzz7rf: ship stamps a permanent 'NOT in trunk — closed anyway' record on every task it lands, because accept (step 1) checks the branch before integrate (step 2) merges it (event 01KZP8RF9JFM9X71ZJGWWHCVN0)
- 2026-08-10T16:47:55Z accepted by a-root (applied 1 proposal(s))
- 2026-08-10T16:47:55Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T16:47:55Z deliverable: no dacli/325-trace-one-user-invoked-verb-end-to-end-across-slice-seams-and-name-where-the branch — nothing to check against trunk
- 2026-08-10T16:47:55Z completed by a-root
