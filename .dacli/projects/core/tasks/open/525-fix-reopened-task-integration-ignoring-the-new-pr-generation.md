---
id: t-01M12K8SEVWQQJXS5MBPMTJWNR
kind: task
created: 2026-08-27T21:50:56Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Fix reopened task integration ignoring the new PR generation
## Acceptance
- [ ] PR resolution prefers an open PR whose head matches the current canonical task branch and generation over a historical merged PR.
- [ ] dacli pr status and dacli integrate select the follow-up PR for a reopened task and do not report already landed from the previous generation.
- [ ] A regression fixture covers a completed task reopened to generation 1 with both a historical merged PR and a current open PR.
- [ ] Existing single-generation and multi-slice PR resolution tests continue to pass, along with go test ./...
## Log
- 2026-08-27T21:53:04Z claimed by a-fixer-zpvnda
