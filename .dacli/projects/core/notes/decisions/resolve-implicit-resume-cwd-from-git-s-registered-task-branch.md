---
id: d-resolve-implicit-resume-cwd-from-git-s-registered-task-branch
kind: note
note_kind: decision
created: 2026-08-16T17:11:54Z
created_by: a-maintainer-gcwx9y
about: "[[t-01KZZVWMD0KYAPDN9QMQDK1GF3]]"
---
# Resolve implicit resume cwd from Git's registered task branch
## Chose
Resolve implicit resume cwd from Git's registered task branch
## Rejected
Infer task ownership from the worktree directory name or always fall back to the main checkout
## Because
Git's worktree registry binds the caller path to the actual task branch; directory names can be copied or renamed, while main-checkout fallback reproduced issue #673
