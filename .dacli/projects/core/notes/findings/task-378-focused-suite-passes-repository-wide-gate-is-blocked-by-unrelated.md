---
id: f-task-378-focused-suite-passes-repository-wide-gate-is-blocked-by-unrelated
kind: note
note_kind: finding
created: 2026-08-12T15:39:34Z
created_by: a-codex-maintainer-vxzmpg
about: "[[378]]"
severity: moderate
---
# Task 378 focused suite passes; repository-wide gate is blocked by unrelated sandbox failures
Verified gofmt -l . clean, go vet ./... passes, and GOCACHE=/private/tmp/dacli-378-gocache go test ./internal/features/orchestration/... passes. Full go test ./... fails outside the claimed slice: internal/cli TestE2EFixtureRepoGoesFromEmptyToShipped child failed, internal/features/execution oversized detached prompt and process-start identity tests failed, and internal/procmon process-table tests observed zero processes. golangci-lint is unavailable (command not found).
