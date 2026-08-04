---
id: d-215-keyed-worktree-path-on-project-seq-slug-in-workspace-worktreepath
kind: note
note_kind: decision
created: 2026-08-04T10:17:52Z
created_by: a-maintainer-mshdxy
about: "[[215]]"
---
# 215: keyed worktree path on project+seq+slug in workspace.WorktreePath
## Chose
215: keyed worktree path on project+seq+slug in workspace.WorktreePath
## Rejected
adding a per-caller composed name string or hashing the task ID
## Because
all 5 callers already hold *store.Task; passing (project, seq, slug) keeps the single collision-free key in workspace (which owns the .dacli layout) and mirrors the branch name dacli/NNN-slug with a project prefix, so the on-disk layout stays greppable per project without a store import into workspace
