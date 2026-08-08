---
id: t-01KZ63A5K91XHDGANBG5FXWY81
kind: task
created: 2026-08-04T09:57:39Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 1, pessimistic: 2}"
---
# doctor reports the loop anchor as orphaned on every run
## So that
doctor output stays signal so nobody learns to ignore it
## Acceptance
- [x] the standing review anchor is not reported as an orphaned task
## Log
- 2026-08-04T11:43:38Z claimed by a-maintainer-as4r9s
- 2026-08-04T11:50:47Z accepted by a-root
- 2026-08-04T11:50:47Z verified by `go test ./internal/features/insight/` (exit 0)
- 2026-08-04T11:50:47Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/309 (event 01KZ69T8K648J0HAW9VKC0P24R)
