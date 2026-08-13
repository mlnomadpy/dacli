---
id: f-loop-claim-and-refusal-regressions-pass-full-verification
kind: note
note_kind: finding
created: 2026-08-13T15:00:12Z
created_by: a-fixer-dd0fvf
about: "[[419]]"
severity: major
---
# Loop claim and refusal regressions pass full verification
internal/store/claimhints_test.go covers task 408's new contracts/controlplane/v1 boundary and rejects internal/features/execution; internal/features/orchestration/driver_test.go covers pre-spawn wave collision filtering and exit-3 blocking. Mutation failures were ClaimHints=[internal/features/execution], all three planned tasks spawned, and rollup produced nothing 1. Green: gofmt -l . empty; GOCACHE=/private/tmp/dacli-419-gocache go vet ./...; GOLANGCI_LINT_CACHE=/private/tmp/dacli-419-lintcache golangci-lint v2.12.2 0 issues; GOCACHE=/private/tmp/dacli-419-gocache go test ./... passed.
