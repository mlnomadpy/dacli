---
id: f-a-loop-that-returns-on-the-first-ref-that-exists-makes-every-later-ref-dead-code
kind: note
note_kind: finding
created: 2026-08-11T16:20:35Z
created_by: a-root
about: "[[[[362]]]]"
severity: major
scope: workspace
origin: internal/store/landing.go
---
# A loop that returns on the first ref that EXISTS makes every later ref dead code
LandingOfRef iterated [origin/<trunk>, refs/heads/<trunk>] and returned LandingUnlanded from inside the loop. The list reads as thorough - two refs consulted - but the early return meant the SECOND ref was unreachable whenever the first existed. And the first is origin/<trunk>, which is stale by construction on the default ship path: ship merges locally, records the verdict, then pushes.

Result: every task shipped through the default path on a repo with a remote got a permanent, committed 'NOT in main - closed anyway' line stamped on work that had just been merged into main. The record is the product, so this is the most expensive class of bug this tool has.

The generalisable shape: a search over candidate sources that returns a VERDICT from the first candidate that RESPONDS, rather than from the first that answers affirmatively. Three ceilings in the seq allocator had the mirror version of the same fault (each covered where its author was looking; a task file that exists but does not parse fell between all three).

Worth grepping for: any 'for _, cand := range ...' whose body returns a negative result unconditionally. It is indistinguishable from correct code at a glance and its later candidates are never executed.
