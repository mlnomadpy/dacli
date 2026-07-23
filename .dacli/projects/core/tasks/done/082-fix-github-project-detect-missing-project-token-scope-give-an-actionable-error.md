---
id: t-01KY5Y1YXV359WJQW2BSGJDMRW
kind: task
created: 2026-07-22T22:10:05Z
created_by: a-root
owner: a-root
priority: could
estimate: {optimistic: 1, probable: 2, pessimistic: 3}
---
# FIX github project: detect missing project token scope, give an actionable error
## Acceptance
- [x] dacli github project detects gh's 'unknown owner type' (missing project scope) and tells the operator to run 'gh auth refresh -s project' instead of surfacing gh's cryptic message
- [x] committed by an agent and opened as a PR; go build + go test ./internal/... green
## Log
- 2026-07-22T22:10:05Z claimed by a-aztk8559eb
- 2026-07-22T22:17:32Z accepted by a-root
- 2026-07-22T22:17:32Z completed by a-root
- 2026-07-22T22:32:13Z accepted by a-root
- 2026-07-22T22:32:13Z completed by a-root
- 2026-07-22T23:52:35Z a-aztk8559eb: PR opened: https://github.com/mlnomadpy/dacli/pull/43 (event 01KY5YA2SJG4QSC980R7ZFJ1MR)
