---
id: f-broken-worktree-git-links-make-prune-preview-and-apply-disagree
kind: note
note_kind: finding
created: 2026-08-13T23:08:11Z
created_by: a-fixer-7w7rgg
about: "[[439]]"
severity: major
---
# Broken worktree git links make prune preview and apply disagree
internal/gitx/gitx.go:298 previously returned immediately when git worktree remove refused a checkout whose .git link was missing; internal/cli/lifecycle_test.go reproduces preview reporting one finished run followed by apply reclaiming zero.
