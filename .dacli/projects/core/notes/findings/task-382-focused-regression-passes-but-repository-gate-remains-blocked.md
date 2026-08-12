---
id: f-task-382-focused-regression-passes-but-repository-gate-remains-blocked
kind: note
note_kind: finding
created: 2026-08-12T18:28:10Z
created_by: a-codex-maintainer-j8jbvt
about: "[[382]]"
severity: moderate
---
# task 382 focused regression passes but repository gate remains blocked
gofmt -l . and go vet ./... pass; go test ./... passes internal/features/execution but fails only internal/cli TestE2EFixtureRepoGoesFromEmptyToShipped at e2e_fixture_test.go:93 on the known spawned-worker sandbox failure; golangci-lint is not installed. Red proof before fix: runrecord_test.go TestStatusReadsDoNotFinalizeALiveRunWhenProcessIdentityIsHidden reported 'status reads finalized a run whose guardian is alive' with outcome no visible result.
