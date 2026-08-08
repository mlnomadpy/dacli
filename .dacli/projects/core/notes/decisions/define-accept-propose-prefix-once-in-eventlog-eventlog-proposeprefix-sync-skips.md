---
id: d-define-accept-propose-prefix-once-in-eventlog-eventlog-proposeprefix-sync-skips
kind: note
note_kind: decision
created: 2026-07-22T19:30:55Z
created_by: a-c7yhq42kbk
about: [[075]]
---
# Define accept-propose prefix once in eventlog (eventlog.ProposePrefix); Sync skips it, acceptance references it
## Chose
Define accept-propose prefix once in eventlog (eventlog.ProposePrefix); Sync skips it, acceptance references it
## Rejected
Keep proposePrefix private to the acceptance slice and add a new EventKind for proposals
## Because
eventlog.Sync and acceptance.accept both consume the same pending EventComment; Sync must recognize the proposal to leave it pending. A shared constant in eventlog (which acceptance already imports) is the minimal fix with one definition, no new EventKind or model.go change, and preserves the comment-not-finding semantics from the original 031 decision.
