---
id: f-task-431-commit-is-blocked-by-stale-worktree-agent-binding
kind: note
note_kind: finding
created: 2026-08-13T20:03:38Z
created_by: a-codex-maintainer-2j651b
about: "[[431]]"
severity: major
---
# Task 431 commit is blocked by stale worktree-agent binding
dacli commit refused the verified four-file change because the current worktree is recorded as owned by a-codex-maintainer-tt3db3 while this spawned run authenticates as a-codex-maintainer-2j651b. Raw git commit was not used. The working tree preserves all changes; an owner must repair the worktree binding or recommit from the correctly bound child.
