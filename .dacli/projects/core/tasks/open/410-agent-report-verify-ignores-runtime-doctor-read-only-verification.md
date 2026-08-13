---
id: t-01KZXER4FMD6JYSPJVK7TBNJEZ
kind: task
created: 2026-08-13T11:40:02Z
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
- [ ] After runtime doctor verifies cc and codex-ro, verify no longer reports their sandbox probe as unknown
- [ ] verify hydrates runtime probe state through the shared store boundary instead of duplicating provider-specific logic
- [ ] A regression test fails when verify loads the raw unhydrated runtime and passes with the shared hydrated runtime
- [ ] gofmt, go vet, pinned golangci-lint, and go test ./... pass
## Log
