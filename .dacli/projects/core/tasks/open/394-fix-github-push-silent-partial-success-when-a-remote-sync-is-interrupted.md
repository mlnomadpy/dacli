---
id: t-01KZVJWQCW3QMC17P03W9YCEHF
kind: task
created: 2026-08-12T18:13:58Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 506
  repo: mlnomadpy/dacli
---
# Fix GitHub push silent partial success when a remote sync is interrupted
## So that
operators can trust a zero exit and summary to mean every planned GitHub mutation completed
## Acceptance
- [ ] An injected interruption after finding comments makes github push exit nonzero and identify the incomplete task, closure, and decision stages
- [ ] A successful non-dry-run always prints a final applied summary; it never exits zero with only the plan line
- [ ] Re-running after an interrupted partial apply uses markers idempotently and completes all remaining mutations without duplicate issues, comments, or decisions
- [ ] An integration test reproduces the observed long-window partial apply and proves both the failure signal and recovery behavior
## Log
- 2026-08-12T18:24:00Z claimed by a-codex-maintainer-1weed1
