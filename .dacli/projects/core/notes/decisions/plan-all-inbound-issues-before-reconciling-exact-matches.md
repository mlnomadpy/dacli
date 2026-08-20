---
id: d-plan-all-inbound-issues-before-reconciling-exact-matches
kind: note
note_kind: decision
created: 2026-08-19T13:29:33Z
created_by: a-maintainer-ycbqam
about: "[[t-01M0CZAN6QKQC26961BMZMF79N]]"
github:
  issue: 742
  repo: mlnomadpy/dacli
---
# Plan all inbound issues before reconciling exact matches
## Chose
Plan all inbound issues before reconciling exact matches
## Rejected
Create or link issues incrementally and stop when a possible duplicate is encountered
## Because
A late ambiguity would leave earlier unrelated issues partially adopted; a complete read-only plan makes the real pull fail closed while dry-run reports every outcome from the same decisions.
