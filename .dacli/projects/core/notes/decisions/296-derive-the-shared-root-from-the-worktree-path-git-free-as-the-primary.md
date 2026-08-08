---
id: d-296-derive-the-shared-root-from-the-worktree-path-git-free-as-the-primary
kind: note
note_kind: decision
created: 2026-08-04T20:47:48Z
created_by: a-maintainer-tcn4vf
about: "[[296]]"
---
# 296: derive the shared root from the worktree PATH (git-free) as the primary redirect, keep git rev-parse as fallback
## Chose
296: derive the shared root from the worktree PATH (git-free) as the primary redirect, keep git rev-parse as fallback
## Rejected
keep relying solely on git rev-parse --path-format=absolute --git-common-dir at every invocation
## Because
the git call silently returns "" on any error (old git without --path-format=absolute, or a sandbox that does not allowlist git), and Find then resolves the stale worktree .dacli shadow -> cryptic 'agent token not recognized'. dacli always creates worktrees at <root>/.dacli/worktrees/<name>, so the shared root is a deterministic prefix of the path with no subprocess; this makes resolution not-dependent-on-which-.dacli-shadows-which and immune to git version/sandbox.
