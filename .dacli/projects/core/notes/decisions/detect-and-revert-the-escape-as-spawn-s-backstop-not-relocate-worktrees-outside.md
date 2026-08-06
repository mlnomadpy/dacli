---
id: d-detect-and-revert-the-escape-as-spawn-s-backstop-not-relocate-worktrees-outside
kind: note
note_kind: decision
created: 2026-08-05T13:27:45Z
created_by: a-fixer-zrcq2v
about: "[[302]]"
---
# Detect-and-revert the escape as spawn's backstop, not relocate worktrees outside the main checkout
## Chose
Detect-and-revert the escape as spawn's backstop, not relocate worktrees outside the main checkout
## Rejected
Moving .dacli/worktrees/ to a path outside w.Root so 'the repo' can no longer mean main
## Because
cmd.Dir is already correctly the worktree and the brief already forbids editing outside it (both cooperative, RUNTIMES.md § 8) -- a real incident today (a-root, unmerged branch dacli/302 at e850f9a) showed the child still wrote a file into main despite both, so the gap isn't cwd or the preamble being wrong, it's that neither is enforced. Relocating the worktree directory doesn't fix that either: an agent can still reference an absolute path into main regardless of where its own worktree physically lives, and it touches worktree prune, the dacli-215 path keying, and every recorded run path for no actual enforcement gain -- a-root filed it as its own follow-up task and never created it. Spawn --worktree now snapshots the main checkout's dirty paths (gitx.DirtyPaths, tracked+untracked, excluding .dacli) right before the child runs and right after, and reverts anything new before returning -- git checkout -- for a tracked file, remove for an untracked one. This makes 'cannot modify the main checkout' true as an observable postcondition without the larger migration, at the cost of being detect-and-revert rather than prevention: it does not stop the write mid-flight and does not catch a child that commits directly into main's git history instead of dirtying its working tree.
