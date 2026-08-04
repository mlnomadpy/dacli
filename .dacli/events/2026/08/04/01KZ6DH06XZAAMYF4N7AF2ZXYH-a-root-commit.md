---
id: 01KZ6DH06XZAAMYF4N7AF2ZXYH
kind: event
event_kind: commit
created: 2026-08-04T12:56:08Z
created_by: a-root
origin: agent
applied: false
---
77b55c7 200: declare max_points across the roster so capacity routing has something to rank

Only 3 of 18 roles carried a cap (fixer 8, prompt-auditor 8, junior 3).
CheapestCapable ranks on model tier then capacity, so with 15 roles
declaring nothing the capacity half had almost no signal and every task
fell through to junior — the cheapest role that happened to have a cap
at all. That is the mechanism behind 238's symptom, and it is also how
work got routed to the one role whose runtime cannot write (250).

Caps are in the same unit as the backlog's Te, which has run 0.2 to 5.5
on this project: personas and estimator 2 (one bounded artifact per
run), junior 3 unchanged, integrator / frontend-reviewer /
ui-ux-designer / role-architect 6, go-auditor / frontend-engineer /
ux-researcher / visionary 8, fixer and prompt-auditor 8 unchanged,
maintainer and reviewer 12.

The two 12s are deliberate. Those roles have no natural ceiling, and a
cap they never hit still beats no cap: an ABSENT cap makes a role
invisible to the ranking rather than expensive in it, which is the
inversion that produced the junior-for-everything answer.

Scope is deliberately untouched in the same pass — scope feeds claim
enforcement, and a wrong glob silently narrows what an agent may commit.
role-architect also gains the summary it was missing, which is what
`team list` and the generated ROSTER print.

Measured after: `team assign 200` (Te 4.3) now answers fixer instead of
junior, because junior's cap of 3 excludes it. The ranking discriminates
for the first time.

The prompts themselves needed nothing — all 18 already carry a Method
section of 40+ lines. The task's title described a gap that had been
closed; the real gap was the metadata that makes routing work.
role: root
