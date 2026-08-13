---
id: d-keep-replay-checkpoints-per-event-and-checksum-only-immutable-event-payload
kind: note
note_kind: decision
created: 2026-08-13T19:57:42Z
created_by: a-codex-maintainer-zkfgn1
about: "[[430]]"
github:
  issue: 615
  repo: mlnomadpy/dacli
---
# Keep replay checkpoints per event and checksum only immutable event payload
## Chose
Keep replay checkpoints per event and checksum only immutable event payload
## Rejected
Add one global replay cursor or include the mutable applied marker in the checksum
## Because
Per-event applied markers already survive partial ownership and skipped events; a global cursor could skip unresolved events, while excluding applied lets the checkpoint advance without invalidating payload integrity
