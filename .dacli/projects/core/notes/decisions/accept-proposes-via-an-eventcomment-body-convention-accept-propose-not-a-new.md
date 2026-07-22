---
id: d-accept-proposes-via-an-eventcomment-body-convention-accept-propose-not-a-new
kind: note
note_kind: decision
created: 2026-07-22T14:50:28Z
created_by: a-yqfsr7b052
about: [[031]]
---
# accept proposes via an EventComment body convention (accept-propose:), not a new event kind
## Chose
accept proposes via an EventComment body convention (accept-propose:), not a new event kind
## Rejected
add a new EventKind (e.g. propose-check) applied by eventlog/sync.go
## Because
model.go and eventlog/sync.go are outside 031's STRICT scope (only acceptance/, store.go, cli.go); an EventComment carrying the accept-propose: prefix is the minimal-but-real event a read-only child emits and dacli accept applies. A comment (not a finding) is the right semantics: a proposed close is an intention, not a discovered fact, so it must not create a durable finding note.
