---
id: d-persist-verified-inbound-envelopes-before-advancing-the-cursor
kind: note
note_kind: decision
created: 2026-08-13T16:13:34Z
created_by: a-codex-maintainer-sg0bxk
about: "[[425]]"
---
# Persist verified inbound envelopes before advancing the cursor
## Chose
Persist verified inbound envelopes before advancing the cursor
## Rejected
Persist only event and idempotency indexes in the cursor
## Because
The v1 protocol defines an append-only Inbox; committing the immutable envelope first makes crash retry repairable while signature and version refusals remain write-free.
