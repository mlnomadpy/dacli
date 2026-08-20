---
id: f-verification-provenance-fix-committed-but-pr-blocked-by-claim-scoped-regression
kind: note
note_kind: finding
created: 2026-08-19T13:29:14Z
created_by: a-fixer-z2ed21
about: "[[t-01M0AGCX64GETGKKBBJK5SKG7D]]"
severity: major
---
# Verification provenance fix committed but PR blocked by claim-scoped regression
Commit a26f04d changes acceptance/planning/store verification to execute and resolve Git from ctx.Cwd and adds the non-Git provenance test. The public command-table regression must live at internal/cli/verification_worktree_test.go, but dacli commit refused it outside the claim, so it was removed. task check is owner-refused. Focused tests/race tests and go test ./... were run with GOCACHE under /tmp; golangci-lint is unavailable (command not found).
