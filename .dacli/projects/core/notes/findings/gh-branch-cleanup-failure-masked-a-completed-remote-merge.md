---
id: f-gh-branch-cleanup-failure-masked-a-completed-remote-merge
kind: note
note_kind: finding
created: 2026-08-12T19:07:17Z
created_by: a-codex-maintainer-2pkfj7
about: "[[396]]"
severity: major
---
# gh branch cleanup failure masked a completed remote merge
internal/features/vcs/lifecycle.go treated every non-zero gh pr merge --delete-branch result as merge failure. When GitHub merged first but local deletion hit an attached worktree, cmdIntegrate printed integrated 0 and wrote no landing event; TestIntegratePRReportsRemoteMergeWhenWorktreeBlocksGHBranchDeletion reproduces the exact partial-success response.
