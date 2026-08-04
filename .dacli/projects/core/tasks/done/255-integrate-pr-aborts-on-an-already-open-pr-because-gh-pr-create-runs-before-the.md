---
id: t-01KZ67PSA0SMMXFC6ZYSCBQ5YV
kind: task
created: 2026-08-04T11:14:26Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# integrate --pr aborts on an already-open PR because gh pr create runs before the merge gate
## So that
the integrator can merge the PRs the loop already opened, which is every PR it will ever see
## Acceptance
- [x] an existing open PR is detected and reused instead of re-created
- [x] the --auto queue and the check-gated merge are both reached when the PR already exists
- [x] a regression test covers the already-exists path
## Log
- 2026-08-04T11:33:45Z accepted by a-root
- 2026-08-04T11:33:45Z verified by `go test ./internal/features/vcs/` (exit 0)
- 2026-08-04T11:33:45Z completed by a-root
