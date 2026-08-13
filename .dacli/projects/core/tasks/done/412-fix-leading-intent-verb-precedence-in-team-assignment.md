---
id: t-01KZXJP37NE1GGGT3M4CAWYAXX
kind: task
created: 2026-08-13T12:48:50Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 564
  repo: mlnomadpy/dacli
---
# Fix leading intent verb precedence in team assignment
## Acceptance
- [x] A title beginning with Fix routes to an implementer even when later words include verify, audit, or review.
- [x] Titles beginning with Verify, Audit, or Review continue to route to a reviewer.
- [x] Routing tests demonstrate failure when a later keyword is allowed to override the leading intent verb.
- [x] The routing fallback for titles without a recognized leading intent verb remains deterministic and covered by tests.
- [x] gofmt -l ., go vet ./..., golangci-lint run, and go test ./... pass.
## Log
- 2026-08-13T12:49:53Z claimed by a-fixer-2hbsam
- 2026-08-13T12:56:02Z accepted by a-root
- 2026-08-13T12:56:02Z verified by `GOCACHE=/private/tmp/dacli-412-accept-gocache go test ./...` (exit 0) in branch main at 38d9bb4 — proves that tree builds, not that the work is in trunk
- 2026-08-13T12:56:02Z completed by a-root
- 2026-08-13T13:02:47Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/565 (event 01KZXK3V7TCK3VCXCQS6ECJFCA)
- 2026-08-13T13:02:47Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/565 at merge commit b662579f17716a758acd2b36f2c584c158df6b70 into main; local cleanup complete (event 01KZXKEJYYHXQ5W6FDQ48X96C0)
