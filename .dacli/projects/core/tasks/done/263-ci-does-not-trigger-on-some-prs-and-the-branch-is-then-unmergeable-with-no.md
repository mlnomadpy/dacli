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
- [x] the cause of the missing pull_request trigger is identified, or a workflow_dispatch fallback exists
- [x] integrate names a PR with no checks as needing attention rather than leaving it silent
## Log
- 2026-08-04T13:04:37Z claimed by a-maintainer-6qe57k
- 2026-08-04T14:35:59Z accepted by a-root
- 2026-08-04T14:35:59Z verified by `go test ./internal/features/vcs/ ./internal/features/onboard/` (exit 0)
- 2026-08-04T14:35:59Z completed by a-root
