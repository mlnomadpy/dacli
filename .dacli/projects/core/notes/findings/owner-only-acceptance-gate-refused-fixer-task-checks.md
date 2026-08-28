---
id: f-owner-only-acceptance-gate-refused-fixer-task-checks
kind: note
note_kind: finding
created: 2026-08-28T15:14:44Z
created_by: a-fixer-wskwrc
about: "[[t-01M14ABSH137KVYGMWBY9FC927]]"
severity: minor
---
# Owner-only acceptance gate refused fixer task checks
After focused tests passed, dacli task check t-01M14ABSH137KVYGMWBY9FC927 --n 1 was refused: only a-root checks acceptance boxes. Verification evidence: the root-context supervise regression passes; mutating resolveSuperviseWorkDir's transferred-worktree branch caused spawn_worktree_test.go:166 to fail opening the absent task-worktree runtime record.
