---
id: d-use-one-provider-neutral-routing-strategy-with-hard-gate-explanations-before
kind: note
note_kind: decision
created: 2026-08-13T14:25:28Z
created_by: a-fixer-ngpzz6
about: "[[407]]"
---
# Use one provider-neutral routing Strategy with hard-gate explanations before measured ranking
## Chose
Use one provider-neutral routing Strategy with hard-gate explanations before measured ranking
## Rejected
Keep CheapestCapable selection and add separate runtime-specific checks in the loop
## Because
A single explanation-bearing strategy prevents eligibility and ranking from drifting while preserving execution's final launch gates
