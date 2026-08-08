---
id: d-hand-authored-dacli-roles-estimator-md-directly-in-this-worktree-instead-of
kind: note
note_kind: decision
created: 2026-08-04T00:16:01Z
created_by: a-fixer-88dk10
about: "[[229]]"
---
# hand-authored .dacli/roles/estimator.md directly in this worktree instead of running dacli role add
## Chose
hand-authored .dacli/roles/estimator.md directly in this worktree instead of running dacli role add
## Rejected
running 'dacli role add estimator --kind planner --grant rw ...' as normal
## Because
role add writes via w.Root, which workspace.Find redirects to the MAIN checkout for a linked worktree (see filed finding on this task) -- it wrote the file into the main checkout's working directory on branch main instead of onto this task's feature branch, where it would sit as a stray uncommitted file nobody would think to commit or PR. Hand-authoring the file in this worktree's own .dacli/roles/, with frontmatter matching CreateRole's exact field order/shape (verified by parsing it back with mdstore.ReadFile), gets the role file into this branch's git history the normal way -- staged, committed via dacli commit, and shipped through the PR like any other role file (fixer.md, reviewer.md, etc. all live in git history the same way).
