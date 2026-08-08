---
id: t-01KZ4WXGGYXZT711V13E33VWP1
kind: task
created: 2026-08-03T22:46:38Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Add an estimator role that sizes tasks so difficulty is derived not typed
## So that
a backlog is sized by judgment against the real codebase instead of a human guessing three numbers
## Acceptance
- [x] a PM-style role reads a task plus the codebase map and writes a three-point estimate via task estimate
- [x] the estimator records why it sized the task that way
## Log
- 2026-08-04T00:06:46Z claimed by a-fixer-88dk10
- 2026-08-04T00:27:45Z accepted by a-root
- 2026-08-04T00:27:45Z verified by `go build ./...` (exit 0)
- 2026-08-04T00:27:45Z completed by a-root
- 2026-08-04T00:37:40Z a-fixer-88dk10: PR opened: https://github.com/mlnomadpy/dacli/pull/280 (event 01KZ523H56VQNKY062PVD2FM7F)
