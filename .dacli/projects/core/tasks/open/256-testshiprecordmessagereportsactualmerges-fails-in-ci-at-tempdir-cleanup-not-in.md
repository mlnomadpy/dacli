---
id: t-01KZ686A92KZQ0QHKMW1T15BFR
kind: task
created: 2026-08-04T11:22:55Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# TestShipRecordMessageReportsActualMerges fails in CI at TempDir cleanup, not in the assertion
## So that
a green PR is not held up by a cleanup race that has nothing to do with the change
## Acceptance
- [ ] the test tears down every git worktree and subprocess it created before t.TempDir cleanup runs
- [ ] the cause is named in a comment rather than the flake being papered over with a retry
## Log
