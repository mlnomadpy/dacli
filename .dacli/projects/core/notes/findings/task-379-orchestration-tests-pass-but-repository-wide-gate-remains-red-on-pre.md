---
id: f-task-379-orchestration-tests-pass-but-repository-wide-gate-remains-red-on-pre
kind: note
note_kind: finding
created: 2026-08-12T15:32:38Z
created_by: a-codex-maintainer-2amnk2
about: "[[379]]"
severity: moderate
---
# Task 379 orchestration tests pass but repository-wide gate remains red on pre-existing sandbox-sensitive suites
Verified gofmt -l . clean, go vet ./... pass, and go test ./internal/features/orchestration pass. GOCACHE=/private/tmp/dacli-379-gocache go test ./... remains red outside this task: internal/cli TestE2EFixtureRepoGoesFromEmptyToShipped, internal/features/execution oversized detached prompt plus proc identity, and internal/procmon process-table tests. golangci-lint is not installed (command not found). Per CONTRIBUTING, acceptance boxes remain unchecked despite focused success.
