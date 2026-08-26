---
id: d-use-one-store-recovery-lease-predicate-for-audited-task-takeover-and-doctor
kind: note
note_kind: decision
created: 2026-08-26T12:53:34Z
created_by: a-fixer-ertmrt
about: "[[t-01M0Z1C74YZY1WKPYNWPCEPZE1]]"
github:
  issue: 781
  repo: mlnomadpy/dacli
---
# Use one store recovery-lease predicate for audited task takeover and doctor
## Chose
Use one store recovery-lease predicate for audited task takeover and doctor
## Rejected
Separate process and transcript checks in each feature command
## Because
The takeover gate and orphan diagnosis must agree on the same live-process and transcript-active evidence.
