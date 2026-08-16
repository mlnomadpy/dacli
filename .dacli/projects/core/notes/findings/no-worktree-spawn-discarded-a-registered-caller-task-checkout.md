---
id: f-no-worktree-spawn-discarded-a-registered-caller-task-checkout
kind: note
note_kind: finding
created: 2026-08-16T17:11:54Z
created_by: a-maintainer-gcwx9y
about: "[[t-01KZZVWMD0KYAPDN9QMQDK1GF3]]"
severity: major
---
# No-worktree spawn discarded a registered caller task checkout
internal/features/execution/execution.go set workDir=w.Root unless --worktree was explicit. The new TestSpawnFromTaskWorktreeRunsAndEditsOnlyThere reproduces this: mutating same-task resolution back to w.Root fails because runtime-branch.txt is absent from the task checkout.
