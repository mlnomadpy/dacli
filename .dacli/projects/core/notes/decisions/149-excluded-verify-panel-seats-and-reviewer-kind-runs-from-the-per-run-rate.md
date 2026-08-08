---
id: d-149-excluded-verify-panel-seats-and-reviewer-kind-runs-from-the-per-run-rate
kind: note
note_kind: decision
created: 2026-07-26T16:51:45Z
created_by: a-w190nhae40
about: [[149]]
---
# 149: excluded verify-panel seats AND reviewer-kind runs from the per-run Rate population, keyed on team.Role.Kind==reviewer via store.LoadRoles
## Chose
149: excluded verify-panel seats AND reviewer-kind runs from the per-run Rate population, keyed on team.Role.Kind==reviewer via store.LoadRoles
## Rejected
Only excluding verify-panel seats (the sole filter store.runRecords applies), or restricting Rate to runs joined to a done task
## Because
The ceiling is the median per-run cost across calibration samples; those samples are done tasks' completing runs. runRecords drops verify seats, and a reviewer (grant ro, never implements) never completes a task so never becomes a sample -- so the ceiling population is implementer-only. Matching it means Rate must drop verify seats AND reviewer-kind runs. Restricting Rate to task-joined runs was rejected because the existing TestAPIBurn (and real burn) counts taskless implementer runs as the day's rate; kind is the principled join, not task presence.
