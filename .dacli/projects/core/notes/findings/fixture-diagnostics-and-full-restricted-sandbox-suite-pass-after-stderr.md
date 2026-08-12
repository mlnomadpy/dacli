---
id: f-fixture-diagnostics-and-full-restricted-sandbox-suite-pass-after-stderr
kind: note
note_kind: finding
created: 2026-08-12T18:28:15Z
created_by: a-codex-maintainer-f85g9w
about: "[[391]]"
severity: major
---
# Fixture diagnostics and full restricted-sandbox suite pass after stderr separation
internal/cli/e2e_fixture_test.go:138 reads the child transcript before TempDir cleanup; its controlled discarded-stderr mutation fails at line 185 with 'worker stderr was discarded: child transcript: <empty>'. internal/features/vcs/git_test.go proves successful git stderr is excluded from plumbing output. Focused E2E passes, gofmt -l . is empty, go vet ./... passes, and env GOCACHE=/private/tmp/dacli-go-cache go test ./... passes. golangci-lint could not run because the binary is not installed.
