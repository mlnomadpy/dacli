---
id: 01KZ6C3DZNPY5FQR3W6JCCD0VM
kind: event
event_kind: commit
created: 2026-08-04T12:31:15Z
created_by: a-root
origin: agent
applied: true
---
d6a92fd record the wave: 183, 223, 224, 244 closed, and 262 filed by the improvement anchor

Task 244 is the loop's continuous-improvement anchor — its deliverable
is one evidence-based task, not a diff — and this run it found a real
dogfooding hole: internal/features/catalog's test drives cmdCatalog
without clearing DACLI_AGENT, so the suite passes in CI and FAILS when
run from inside a dacli agent session with 'agent token not recognized'.
Every sibling command test clears it (insight, planning, acceptance) and
catalog is the one that does not. Filed as 262, test-only, and spawned.

That is the class of defect CI structurally cannot see: CI never runs
the suite as a spawned agent, and this project's agents run it
constantly.
role: root
