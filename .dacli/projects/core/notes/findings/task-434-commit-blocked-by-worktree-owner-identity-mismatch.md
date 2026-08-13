---
id: f-task-434-commit-blocked-by-worktree-owner-identity-mismatch
kind: note
note_kind: finding
created: 2026-08-13T20:03:04Z
created_by: a-codex-maintainer-grz3zz
about: "[[434]]"
severity: major
---
# Task 434 commit blocked by worktree-owner identity mismatch
dacli commit refused: worktree owned by a-codex-maintainer-8c7ncp but active brief/token identifies a-codex-maintainer-grz3zz; refusal says restore that child's DACLI_AGENT token and confirms staged work preserved. Per exit-code contract it was not retried and raw git commit was not used. All Go verification except unavailable golangci-lint had passed before refusal.
