---
id: d-integrate-tasks-branch-cleanup-deletes-the-merged-branch-inside-mergetask-not
kind: note
note_kind: decision
created: 2026-07-22T14:45:24Z
created_by: a-3gax3zfmsa
about: [[034]]
---
# integrate --tasks branch cleanup deletes the merged branch inside mergeTask, not in a gitx helper
## Chose
integrate --tasks branch cleanup deletes the merged branch inside mergeTask, not in a gitx helper
## Rejected
add a DeleteBranch() to internal/gitx/gitx.go and call it from cmdIntegrate only
## Because
STRICT scope forbids touching gitx.go, and folding branch deletion into the shared mergeTask (via gitx.Run(w.Root, branch -D)) means single-task dacli merge cleans up identically to integrate — one code path, consistent cleanup, no cross-file edit; worktree is removed before the branch since a branch checked out in a worktree cannot be deleted, and the delete is best-effort so a failure never leaves a half-merged tree
