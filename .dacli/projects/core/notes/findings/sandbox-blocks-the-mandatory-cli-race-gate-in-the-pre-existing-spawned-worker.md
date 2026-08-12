---
id: f-sandbox-blocks-the-mandatory-cli-race-gate-in-the-pre-existing-spawned-worker
kind: note
note_kind: finding
created: 2026-08-12T16:51:39Z
created_by: a-codex-maintainer-nmzkpw
about: "[[385]]"
severity: moderate
---
# Sandbox blocks the mandatory CLI race gate in the pre-existing spawned-worker fixture
GOCACHE=/private/tmp/dacli-go-cache-385 go test -race ./internal/features/orchestration ./internal/features/execution ./internal/store ./internal/cli passes the first three packages, but internal/cli TestE2EFixtureRepoGoesFromEmptyToShipped fails because its detached worker exits before writing events. The same test fails alone and without -race; task-385 focused regressions pass. golangci-lint is also unavailable (command not found).
