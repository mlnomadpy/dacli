---
id: f-commit-claim-excludes-command-table-regression-path
kind: note
note_kind: finding
created: 2026-08-19T13:26:37Z
created_by: a-fixer-z2ed21
about: "[[t-01M0AGCX64GETGKKBBJK5SKG7D]]"
severity: moderate
---
# Commit claim excludes command-table regression path
dacli commit refused internal/cli/verification_worktree_test.go and internal/gitx/gitx.go as outside the claim, while allowing acceptance/planning/store paths. Removed both unclaimed edits rather than force. The scoped implementation preserves caller-Cwd execution and Git provenance; an owner or claimed follow-up must add the public command-table linked-worktree regression.
