---
id: d-land-selected-pr-tasks-before-final-acceptance-inside-ship
kind: note
note_kind: decision
created: 2026-08-28T13:42:14Z
created_by: a-maintainer-wg7dnx
about: "[[t-01M12QX9HEPKAAS1033W6HS45D]]"
---
# Land selected PR tasks before final acceptance inside ship
## Chose
Land selected PR tasks before final acceptance inside ship
## Rejected
Keep accept-before-integrate with deferred landing
## Because
PR merge/check failure must leave the task nonterminal, and final acceptance/issue reconciliation must follow observed remote merge, fresh-base identity, and post-landing verification rather than temporarily closing before the remote transaction.
