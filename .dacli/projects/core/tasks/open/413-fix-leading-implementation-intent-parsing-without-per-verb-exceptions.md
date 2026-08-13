---
id: t-01KZXKHGDD0JJ1GK11MF5F26TG
kind: task
created: 2026-08-13T13:03:48Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 567
  repo: mlnomadpy/dacli
---
# Fix leading implementation intent parsing without per-verb exceptions
## Acceptance
- [ ] Titles led by Test, Check, Improve, and Cover route to implementers when the second word is verify, audit, or review.
- [ ] Supported modifier forms such as Full audit continue to route to reviewers.
- [ ] The parser uses an explicit leading-intent or modifier rule instead of adding individual implementation verbs to the reviewer keyword table.
- [ ] A regression test fails against merge b662579 before the fix and passes afterward.
- [ ] gofmt -l ., go vet ./..., pinned golangci-lint run, and go test ./... pass.
## Log
