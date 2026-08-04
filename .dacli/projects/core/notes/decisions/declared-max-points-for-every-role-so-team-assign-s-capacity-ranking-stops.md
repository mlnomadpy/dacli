---
id: d-declared-max-points-for-every-role-so-team-assign-s-capacity-ranking-stops
kind: note
note_kind: decision
created: 2026-08-04T12:35:31Z
created_by: a-root
---
# Declared max_points for every role, so team assign's capacity ranking stops being vacuous
## Chose
Declared max_points for every role, so team assign's capacity ranking stops being vacuous
## Rejected
leaving caps undeclared and routing on model tier alone
## Because
Only 3 of 18 roles declared max_points (fixer 8, prompt-auditor 8, junior 3), so the capacity half of CheapestCapable had almost nothing to rank and every task fell through to junior — the cheapest role that happened to have a cap. That is the mechanism behind task 238's symptom, and it also routed work to the one role whose runtime cannot write (task 250). The caps below are in the same unit as the backlog's Te, where this project's tasks have run 0.2 to 5.5: junior 3 (unchanged, cheap model, escalates early), estimator 2 and each persona-* 2 (one bounded artifact per run), integrator 6, frontend-reviewer 6, ui-ux-designer 6, role-architect 6, fixer 8 (unchanged), prompt-auditor 8 (unchanged), go-auditor 8, frontend-engineer 8, ux-researcher 8, visionary 8, maintainer 12, reviewer 12. The two 12s are deliberate: maintainer and reviewer are the roles with no natural ceiling, and a cap they never hit is still better than no cap, because an absent cap makes them invisible to the ranking rather than expensive in it. Scope is deliberately NOT changed in the same pass — scope feeds claim enforcement, and a wrong glob silently narrows what an agent may commit.
