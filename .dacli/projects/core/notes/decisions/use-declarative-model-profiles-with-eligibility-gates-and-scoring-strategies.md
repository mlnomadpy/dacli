---
id: d-use-declarative-model-profiles-with-eligibility-gates-and-scoring-strategies
kind: note
note_kind: decision
created: 2026-08-13T09:37:24Z
created_by: a-root
about: "[[403]]"
github:
  issue: 555
  repo: mlnomadpy/dacli
---
# Use declarative model profiles with eligibility gates and scoring strategies
## Chose
Use declarative model profiles with eligibility gates and scoring strategies
## Rejected
Hardcoded provider model names and one global cheapest-model sort
## Because
Provider names and prices change, while capability, capacity, cost tier, context, grant enforcement, and measured outcomes are stable routing concepts; hard gates must run before ranking so a cheap but incapable model can never win.
