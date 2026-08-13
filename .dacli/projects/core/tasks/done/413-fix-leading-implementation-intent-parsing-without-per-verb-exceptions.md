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
- [x] Titles led by Test, Check, Improve, and Cover route to implementers when the second word is verify, audit, or review.
- [x] Supported modifier forms such as Full audit continue to route to reviewers.
- [x] The parser uses an explicit leading-intent or modifier rule instead of adding individual implementation verbs to the reviewer keyword table.
- [x] A regression test fails against merge b662579 before the fix and passes afterward.
- [x] gofmt -l ., go vet ./..., pinned golangci-lint run, and go test ./... pass.
## Log
- 2026-08-13T13:04:24Z claimed by a-fixer-rsb99q
- 2026-08-13T13:10:22Z accepted by a-root
- 2026-08-13T13:10:22Z verified by `GOCACHE=/private/tmp/dacli-413-accept-gocache go test ./...` (exit 0) in branch main at b662579 — proves that tree builds, not that the work is in trunk
- 2026-08-13T13:10:22Z completed by a-root
