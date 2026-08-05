---
id: f-spawn-worktree-s-escape-guard-only-covers-the-synchronous-path-detach-and
kind: note
note_kind: finding
created: 2026-08-05T13:27:45Z
created_by: a-fixer-zrcq2v
about: "[[302]]"
severity: moderate
---
# spawn --worktree's escape guard only covers the synchronous path; --detach and supervise/verify are unguarded
The dirty-diff-and-revert backstop I added lives in cmdSpawn's non-detach branch (internal/features/execution/execution.go, right after the execRuntime call at ~line 684) because that's where there's a natural 'child just exited' point to check from. --detach returns before the child even starts working, so the same guard there would need to live in 'dacli wait's finalization instead -- not done here, out of scope for this task's acceptance criteria. Separately (already known, not new): 'supervise' and 'verify' have no --worktree flag at all (execution.go:904, verify.go:148 both hardcode w.Root as cwd), so every multi-turn supervised run and every verify panelist always runs with cwd = main checkout, worktree isolation guard or not. Both are real gaps worth their own tasks.
