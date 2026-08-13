---
id: f-verified-task-430-changes-are-staged-but-commit-is-blocked-by-worktree-identity
kind: note
note_kind: finding
created: 2026-08-13T20:08:11Z
created_by: a-codex-maintainer-zkfgn1
about: "[[430]]"
severity: major
---
# Verified task 430 changes are staged but commit is blocked by worktree identity mismatch
dacli commit returned exit 3: worktree owned by a-codex-maintainer-q0y479 but active brief/run identity is a-codex-maintainer-zkfgn1. Per policy no retry or raw git commit was attempted. Four internal/eventlog files remain staged; gofmt, go vet ./..., go test ./..., focused -race, and checksum mutation proof completed.
