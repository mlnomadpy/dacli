---
id: d-bind-root-recovery-claims-to-checkout-plus-branch-or-task-context
kind: note
note_kind: decision
created: 2026-08-27T23:59:58Z
created_by: a-maintainer-ve10nq
about: "[[t-01M0N00V8CYZ3S125G5HJ2CTYN]]"
---
# Bind root recovery claims to checkout plus branch or task context
## Chose
Bind root recovery claims to checkout plus branch or task context
## Rejected
Expire or delete completed transfer records
## Because
Transfers are durable attribution evidence and must survive restart; contextual authorization excludes unrelated history without erasing it.
