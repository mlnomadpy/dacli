---
id: f-recovered-wait-lifecycle-regression-is-mutation-proven-and-locally-green
kind: note
note_kind: finding
created: 2026-08-13T21:06:10Z
created_by: a-codex-maintainer-8r5s5s
about: "[[436]]"
severity: major
---
# Recovered wait lifecycle regression is mutation-proven and locally green
TestAgentsRecoveryLetsWaitFinalizeMultipleNamedRuns in internal/features/execution/claim_release_test.go calls agents before wait for two named dead runs, verifies both proc.txt claims are released and both outcome.md files finalize. Removing the rec.Outcome terminal guard at execution.go:2852 fails with: wait timed out with 2 run(s) still live. gofmt, go vet, package tests, full go test ./..., and git diff --check pass; golangci-lint is unavailable (command not found).
