---
id: d-299-derive-the-build-spawn-s-claim-from-task-pathhints-the-same-extraction
kind: note
note_kind: decision
created: 2026-08-06T08:23:43Z
created_by: a-fixer-43t0j0
about: "[[299]]"
---
# 299: derive the build spawn's --claim from Task.PathHints (the same extraction routing already uses), skip the flag when a task names no path
## Chose
299: derive the build spawn's --claim from Task.PathHints (the same extraction routing already uses), skip the flag when a task names no path
## Rejected
a bespoke path extraction, or always passing --claim even when empty
## Because
PathHints already backs team.CheapestCapable's tie-break so the loop's own routing and its --claim agree on what 'the task's files' means; splitClaims treats an empty value as no claim anyway, so appending it unconditionally would be pure noise on every task whose text names nothing path-like
