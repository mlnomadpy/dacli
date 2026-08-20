---
id: f-caller-worktree-now-binds-verification-execution-and-provenance
kind: note
note_kind: finding
created: 2026-08-19T13:25:28Z
created_by: a-fixer-z2ed21
about: "[[t-01M0AGCX64GETGKKBBJK5SKG7D]]"
severity: major
---
# Caller worktree now binds verification execution and provenance
Implemented store.RunVerification(dir, ...) and public-command regression internal/cli/verification_worktree_test.go:17. The test runs task check and accept from a linked branch, asserts pwd there, exact branch/SHA evidence, and shared main .dacli persistence. Mutation: replacing ctx.Cwd with w.Root made task check exit 1 at the pwd assertion. Focused packages, race packages, gofmt, go vet, and go test ./... passed; golangci-lint was unavailable (command not found).
