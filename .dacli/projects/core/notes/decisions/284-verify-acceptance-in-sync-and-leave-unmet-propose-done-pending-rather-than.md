---
id: d-284-verify-acceptance-in-sync-and-leave-unmet-propose-done-pending-rather-than
kind: note
note_kind: decision
created: 2026-08-04T20:13:09Z
created_by: a-maintainer-rgjgzv
about: "[[284]]"
---
# 284: verify acceptance in Sync and leave unmet propose:done pending, rather than record a refusal event
## Chose
284: verify acceptance in Sync and leave unmet propose:done pending, rather than record a refusal event
## Rejected
recording a refusal comment/event when acceptance is unmet, or checking boxes myself
## Because
leaving the propose:done event pending mirrors the owner path's Refusedf exit-3 shape (planning.go cmdTaskDone) and the sibling accept-propose path (sync.go): a human resolves it. It also matches apply()'s existing 'malformed proposal stays pending' branch, so proposedTasks/doctor still surface it. Recording a synthetic refusal event would invent a new event kind out of scope; Sync must not check boxes.
