---
id: f-team-assign-recommends-junior-for-every-open-task-so-cost-routing-is-currently
kind: note
note_kind: finding
created: 2026-08-04T10:14:13Z
created_by: a-root
---
# team assign recommends junior for every open task, so cost routing is currently a no-op
Ran `dacli team assign` over 205, 243, 248, 215, 210 and 183 — six tasks spanning a trivial gitignore change and a subtle GitHub-mirror idempotency bug. All six returned the same recommendation: `dacli spawn --role junior`.

That is the routing working exactly as specified and still being useless. CheapestCapable ranks on model tier first and role capacity second, and junior (haiku, cap 3 points) is capable-by-capacity for every task whose Te is under 3 — which is most of the backlog. Nothing in the ranking knows that 205 touches marker idempotency against a live GitHub repo and 183 edits a gitignore.

This is the empirical case for task 238: assignment needs a scope signal (area, package, blast radius, or an explicit role affinity on the task) as a filter BEFORE cost is used as the tiebreak. Cost-first over a capacity-only capability test collapses to 'always the cheapest role'.

Recorded while spawning this wave; I overrode the recommendation to maintainer on all five non-trivial tasks.
