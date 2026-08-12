---
id: f-task-394-focused-verification-passes-but-the-mandatory-full-suite-remains
kind: note
note_kind: finding
created: 2026-08-12T18:28:25Z
created_by: a-codex-maintainer-1weed1
about: "[[394]]"
severity: moderate
---
# Task 394 focused verification passes but the mandatory full suite remains blocked by the existing E2E spawn failure
gofmt -l . and GOCACHE=/private/tmp/dacli-394-go-cache go vet ./... pass; go test ./... passes every package except internal/cli, where TestE2EFixtureRepoGoesFromEmptyToShipped fails at e2e_fixture_test.go:93 because the worker exits 1 with zero events. internal/features/ghmirror passes. golangci-lint is unavailable on PATH. Per AGENTS.md no acceptance box is checked while the full suite is red.
