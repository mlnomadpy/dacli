---
id: t-01KZ6DSADHNDMQK9J6ZCQW8MNG
kind: task
created: 2026-08-04T13:00:41Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# CI does not trigger on some PRs, and the branch is then unmergeable with no signal
## So that
a PR without checks is treated as unverified rather than quietly waiting forever
## Acceptance
- [ ] the cause of the missing pull_request trigger is identified, or a workflow_dispatch fallback exists
- [ ] integrate names a PR with no checks as needing attention rather than leaving it silent
## Log
