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
- [ ] A title beginning with Fix routes to an implementer even when later words include verify, audit, or review.
- [ ] Titles beginning with Verify, Audit, or Review continue to route to a reviewer.
- [ ] Routing tests demonstrate failure when a later keyword is allowed to override the leading intent verb.
- [ ] The routing fallback for titles without a recognized leading intent verb remains deterministic and covered by tests.
- [ ] gofmt -l ., go vet ./..., golangci-lint run, and go test ./... pass.
## Log
