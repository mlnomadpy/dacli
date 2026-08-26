---
id: d-separate-stdout-readiness-from-stderr-diagnostics
kind: note
note_kind: decision
created: 2026-08-26T14:04:18Z
created_by: a-fixer-qhwq3c
about: "[[t-01M0NR4J956431ZNW3MBDKCH0H]]"
github:
  issue: 789
  repo: mlnomadpy/dacli
---
# Separate stdout readiness from stderr diagnostics
## Chose
Separate stdout readiness from stderr diagnostics
## Rejected
Share one capped output buffer across both pipes
## Because
A stderr flood beyond the diagnostic cap must remain drainable without delaying the JSONL readiness scanner or contending on its coordination lock.
