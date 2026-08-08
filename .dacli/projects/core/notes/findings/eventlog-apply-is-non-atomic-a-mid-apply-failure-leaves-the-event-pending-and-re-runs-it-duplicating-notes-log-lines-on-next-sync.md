---
id: f-eventlog-apply-is-non-atomic-a-mid-apply-failure-leaves-the-event-pending-and-re-runs-it-duplicating-notes-log-lines-on-next-sync
kind: note
note_kind: finding
created: 2026-07-21T23:09:25Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
---
# eventlog.apply is non-atomic: a mid-apply failure leaves the event pending and re-runs it, duplicating notes / log lines on next sync
sync.go:58 flips 'applied' via MarkApplied only AFTER apply() returns success, but apply() (sync.go:67-115) does several committing side effects first. If a later step fails after an earlier one committed, Sync aborts before MarkApplied and the event stays pending → next sync re-runs from the top: EventClaim (:72-73) appends a SECOND 'claimed by …' Log line; EventFinding (:103) calls CreateNote again, and the collision path (store.go:477) assigns a fresh ULID suffix rather than deduping, writing a spurious DUPLICATE finding note. Make apply idempotent (check-before-write) or mark-applied first / commit atomically.
