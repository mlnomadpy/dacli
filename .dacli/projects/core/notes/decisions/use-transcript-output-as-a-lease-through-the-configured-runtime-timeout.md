---
id: d-use-transcript-output-as-a-lease-through-the-configured-runtime-timeout
kind: note
note_kind: decision
created: 2026-08-14T10:14:03Z
created_by: a-maintainer-j68p78
about: "[[t-01KZZVFWZWP3M2KX52E1FF6CMA]]"
---
# Use transcript output as a lease through the configured runtime timeout
## Chose
Use transcript output as a lease through the configured runtime timeout
## Rejected
Extend the fixed transcript-mtime grace
## Because
Codex may be silent for an unbounded edit or test interval; runtime-exit.txt is durable real-exit evidence and the configured timeout supplies the existing finite upper bound, while legacy records retain the short grace.
