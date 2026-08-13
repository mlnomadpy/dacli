---
id: d-treat-fix-as-an-explicit-implementer-intent-verb
kind: note
note_kind: decision
created: 2026-08-13T12:51:02Z
created_by: a-fixer-2hbsam
about: "[[412]]"
github:
  issue: 566
  repo: mlnomadpy/dacli
---
# Treat Fix as an explicit implementer intent verb
## Chose
Treat Fix as an explicit implementer intent verb
## Rejected
Special-case the first title word before the existing verb map
## Because
One verb table preserves the current modifier window and makes precedence come from the existing first-match scan with the smallest behavior change.
