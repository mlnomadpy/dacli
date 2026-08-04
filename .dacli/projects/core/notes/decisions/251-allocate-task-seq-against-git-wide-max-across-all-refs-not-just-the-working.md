---
id: d-251-allocate-task-seq-against-git-wide-max-across-all-refs-not-just-the-working
kind: note
note_kind: decision
created: 2026-08-04T11:44:09Z
created_by: a-maintainer-88hjw4
about: "[[251]]"
---
# 251: allocate task seq against git-wide max across all refs, not just the working tree
## Chose
251: allocate task seq against git-wide max across all refs, not just the working tree
## Rejected
keep scanning only the working tree, or add a persistent monotonic counter file
## Because
a counter file is itself branch-local (each branch commits its own value) so it collides identically; scanning git log --all clears the ceiling of every seq committed on ANY unmerged branch, is best-effort (returns 0 off-git so non-git tests are unchanged), and reuses the existing seq-lock for intra-tree serialization. Reconciliation of pre-existing collisions is surfaced via a new doctor 'collided-seq' check rather than auto-renumbered, because a seq is a live reference (branch/worktree/PR names) an owner must renumber deliberately.
