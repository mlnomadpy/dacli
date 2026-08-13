---
id: d-treat-task-424-as-an-already-landed-duplicate-of-task-370
kind: note
note_kind: decision
created: 2026-08-13T15:35:41Z
created_by: a-fixer-sfkfzr
about: "[[424]]"
github:
  issue: 589
  repo: mlnomadpy/dacli
---
# Treat task 424 as an already-landed duplicate of task 370
## Chose
Treat task 424 as an already-landed duplicate of task 370
## Rejected
Add another production guard or duplicate regression test
## Because
Current HEAD already shares the live runCycle planning path, returns before AfterCycle, suppresses all checkpoint writes in saveState, and has a repeated-preview whole-.dacli digest invariant with a demonstrated red mutation.
