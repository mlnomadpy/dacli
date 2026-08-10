---
id: 01KZ660TRV10J6PGFC7DJG0HZD
kind: event
event_kind: commit
created: 2026-08-04T10:44:58Z
created_by: a-root
origin: agent
applied: true
---
d28d3df file three findings from the wave: runtime/grant mismatch, cross-branch seq collision, worktree accumulation

250 — role junior declares grant rw and runtime cc, and cc's allowlist
has no Edit or Write. The junior spawned on 183 spent a whole run
discovering it could not edit a file, then tried to invoke a config
skill to widen its own permissions. `team assign` recommends junior for
every open task, so the cheapest route is currently a route that cannot
write.

251 — task seq is allocated by scanning the working tree, and a branch
is a tree the allocator cannot see. Filing these three tasks from main
handed out 250 and 251, both already taken by different tasks on the
branch behind PR #284. That is 209/247 one level up: 247 fixed mutual
exclusion between processes over one tree; nothing arbitrates between
trees. Every --worktree spawn files against its own checkout, so the
swarm makes this the normal case.

252 — 86 live worktrees, 2.2 GB, one per task ever spawned with
--worktree and not one of them reclaimed.
role: root
