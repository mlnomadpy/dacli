---
id: 01KZ67HK22JMERM9HK7BGJ5WHC
kind: event
event_kind: commit
created: 2026-08-04T11:11:36Z
created_by: a-root
origin: agent
applied: false
---
57e72a7 skill+docs: teach the landing path, the role/runtime coupling, and the branch-local workspace

The host-agent skill covered planning, spawning and gates but stopped
short of landing — the one part I got wrong repeatedly today. It now
carries:

- `dacli integrate`/`ship` as the merge path, and why: the merge has to
  be recorded as an event against the task, which raw `gh pr merge`
  skips. Plus the three things that bite — the branch name is a lookup
  key and a mismatch is silently skipped, integrate refuses from any
  branch but --into, and overlapping PRs merge one at a time.
- The `integrator` role, which exists precisely so nobody merges by
  hand, and the one case where --worktree is wrong.
- Role grant and runtime must agree, with junior as the worked example:
  grant rw, runtime cc, no Edit or Write in the allowlist, so an
  implementation task burns a run discovering it cannot write a file.
- team assign is a floor, not an answer — it ranks cost and capacity and
  knows nothing about blast radius.
- The workspace forks with the branch: a task filed on a branch does not
  exist on trunk, and seq numbers collide across branches.

Anti-patterns gain `git add -A` during a wave (it stages siblings'
in-flight edits under your name), merging by hand, and letting worktrees
accumulate.

using-dacli (the spawned-child skill) gains the landing section it never
had: add your own paths, commit --no-add, push, and stop — integrate is
the operator's step and refuses from a worktree anyway.

integrator's own prompt told it to run `gh pr merge`, which its runtime
does not allow. Rewritten around dacli integrate — the same defect the
skill now warns about, in the role whose job was to merge.

README: 77 -> 96 merged PRs.
role: root
