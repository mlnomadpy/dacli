---
id: d-project-ship-acceptance-eligibility-in-the-dry-run-explicit-window-resolver
kind: note
note_kind: decision
created: 2026-08-27T13:10:29Z
created_by: a-fixer-11h4hg
about: "[[t-01KZYREM23X0ADW8MDV26C1H9A]]"
---
# Project ship acceptance eligibility in the dry-run explicit-window resolver
## Chose
Project ship acceptance eligibility in the dry-run explicit-window resolver
## Rejected
Invoke accept itself while rendering the dry-run plan
## Because
Dry-run must not mutate task records or execute verification commands; static pending-proposal checks mirror the accept-before-integrate transition for the requested explicit refs.
