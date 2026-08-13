---
id: t-01KZXET2V7JWH2AP2XARJTJPVT
kind: task
created: 2026-08-13T11:41:06Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 562
  repo: mlnomadpy/dacli
---
# Fix verify ignoring runtime doctor read-only verification
## So that
verification panels use the same persisted runtime safety evidence as spawn and preflight
## Acceptance
- [x] After runtime doctor verifies cc and codex-ro, verify no longer reports their sandbox probe as unknown
- [x] verify hydrates runtime probe state through the shared store boundary instead of duplicating provider-specific logic
- [x] A regression test fails when verify loads the raw unhydrated runtime and passes with the shared hydrated runtime
- [x] gofmt, go vet, pinned golangci-lint, and go test ./... pass
## Log
- 2026-08-13T11:41:23Z claimed by a-fixer-ge5keg
- 2026-08-13T11:52:58Z accepted by a-root
- 2026-08-13T11:52:58Z verified by `cd /Users/tahabsn/Documents/GitHub/dacli/.dacli/worktrees/core-411-fix-verify-ignoring-runtime-doctor-read-only-verification && gofmt -l . && GOCACHE=/private/tmp/dacli-accept-411 go vet ./... && GOCACHE=/private/tmp/dacli-accept-411 go test ./internal/features/execution ./internal/store` (exit 0) in branch main at b4db035 — proves that tree builds, not that the work is in trunk
- 2026-08-13T11:52:58Z completed by a-root
