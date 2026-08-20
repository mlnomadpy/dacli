---
id: f-verification-resolves-git-identity-and-command-directory-from-shared-workspace
kind: note
note_kind: finding
created: 2026-08-19T13:19:19Z
created_by: a-fixer-z2ed21
about: "[[t-01M0AGCX64GETGKKBBJK5SKG7D]]"
severity: major
---
# Verification resolves Git identity and command directory from shared workspace root
Confirmed internal/store/verification.go:40-50 calls gitx against w.Root and sets exec.Command.Dir = w.Root; planning.go:472 and acceptance.go:466 provide no caller Cwd, so linked-worktree verification can record and test main.
