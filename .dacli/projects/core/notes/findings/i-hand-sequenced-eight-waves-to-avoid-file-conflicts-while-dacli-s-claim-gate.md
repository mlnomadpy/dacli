---
id: f-i-hand-sequenced-eight-waves-to-avoid-file-conflicts-while-dacli-s-claim-gate
kind: note
note_kind: finding
created: 2026-08-04T20:42:56Z
created_by: a-root
severity: major
origin: internal/features/execution/execution.go:552
---
# I hand-sequenced eight waves to avoid file conflicts while dacli's claim gate sat unused
Every wave today was grouped by package BY ME — 'these six touch disjoint packages, spawn them; hold those four, they all want execution/'. That is the arbitration `spawn --claim` performs, and I passed --claim exactly zero times until now.

The cost is measurable. The two merge conflicts that needed hand resolution today — 224 against 223 in ghmirror's Commands table, and 261 against 223 in cmdShip — both came out of waves spawned with no claims. dacli could not refuse them because nobody told it what anyone was touching.

Verified live just now. Task 274 was spawned claiming internal/brief; spawning 269 with an overlapping claim was refused:

  path-claim conflict: live agent a-maintainer-me4vk0 already claims "internal/brief"
  and you claim "internal/brief" — narrow your scope, or `dacli wait 01KZ785H1A` first

Names the holder, the path, and both recoveries. I narrowed 269 to prompts+execution and it spawned.

The same applies to selection. `dacli next --parallel 6` now returns a critical-path-ordered set with slack per task, because the estimates landed this morning — I had been picking the wave from my own reading of the backlog instead of asking.

The rule, which is the same one the owner already stated about reaching past dacli for raw gh: if I am doing coordination in my head, dacli probably does it, and doing it in my head means the workspace has no record of the decision and the machinery cannot enforce it.
