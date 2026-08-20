---
id: d-enforce-every-command-table-synopsis-begins-with-its-declared-command-path
kind: note
note_kind: decision
created: 2026-08-19T12:32:04Z
created_by: a-fixer-3pqnc4
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
github:
  issue: 721
  repo: mlnomadpy/dacli
---
# Enforce every command table synopsis begins with its declared command path
## Chose
Enforce every command table synopsis begins with its declared command path
## Rejected
Rely only on missing-argument handler comparisons
## Because
Some commands allow omitted positional arguments and documented handler variants can mask a copied command path; the prefix check detects those drifts while retaining exact handler comparisons.
