---
id: f-historical-root-transfers-globally-fenced-later-worktrees
kind: note
note_kind: finding
created: 2026-08-27T23:59:58Z
created_by: a-maintainer-ve10nq
about: "[[t-01M0N00V8CYZ3S125G5HJ2CTYN]]"
severity: major
---
# Historical root transfers globally fenced later worktrees
internal/features/vcs/vcs.go:501 selected the newest transfer by new_owner alone; readWorktreeTransfer discarded its recorded worktree, branch, and prior-run context, so task 492's completed root transfer constrained task 493.
