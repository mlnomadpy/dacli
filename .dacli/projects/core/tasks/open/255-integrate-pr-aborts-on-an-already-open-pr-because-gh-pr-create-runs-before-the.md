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
- [ ] an existing open PR is detected and reused instead of re-created
- [ ] the --auto queue and the check-gated merge are both reached when the PR already exists
- [ ] a regression test covers the already-exists path
## Log
