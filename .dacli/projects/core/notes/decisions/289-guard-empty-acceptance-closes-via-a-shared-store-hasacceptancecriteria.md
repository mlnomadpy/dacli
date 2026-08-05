---
id: d-289-guard-empty-acceptance-closes-via-a-shared-store-hasacceptancecriteria
kind: note
note_kind: decision
created: 2026-08-04T20:35:02Z
created_by: a-maintainer-wyr06h
about: "[[289]]"
---
# 289: guard empty-acceptance closes via a shared store.HasAcceptanceCriteria predicate applied at each close path, refuse by default with an --allow-unverified escape that stamps UNVERIFIED
## Chose
289: guard empty-acceptance closes via a shared store.HasAcceptanceCriteria predicate applied at each close path, refuse by default with an --allow-unverified escape that stamps UNVERIFIED
## Rejected
Guard once inside store.CloseTask
## Because
CloseTask is the canonical close for task done + accept, but the propose-then-sync route closes via store.MoveTask (sync.go:161), NOT CloseTask (that re-route is task 284's job), so a CloseTask-only guard would miss the sync path the acceptance explicitly names. A shared read-only predicate consulted at each path (planning cmdTaskDone, acceptance acceptOne/acceptAll, eventlog sync EventProposeStatus->done) keeps the rule identical on all three without a slice cross-import, and leaves the sync proposal pending (like the malformed-status case) rather than auto-closing.
