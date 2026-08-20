---
id: d-use-one-validated-dependency-event-for-direct-edits-and-owner-sync
kind: note
note_kind: decision
created: 2026-08-20T08:32:17Z
created_by: a-maintainer-d7gr0n
about: "[[t-01M0CZANAQKP50AWEN2C6C8VXR]]"
github:
  issue: 758
  repo: mlnomadpy/dacli
---
# Use one validated dependency event for direct edits and owner sync
## Chose
Use one validated dependency event for direct edits and owner sync
## Rejected
Add a direct-only task rewrite command or encode proposals as generic comments
## Because
A typed mailbox event preserves audit identity, gives read-only agents a proposal path, and makes owner synchronization replay the same store validation idempotently
