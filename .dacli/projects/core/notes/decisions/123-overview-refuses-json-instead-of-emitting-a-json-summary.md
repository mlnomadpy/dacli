---
id: d-123-overview-refuses-json-instead-of-emitting-a-json-summary
kind: note
note_kind: decision
created: 2026-07-23T19:53:43Z
created_by: a-fgmsw5w6rd
about: [[123]]
---
# 123: overview refuses --json instead of emitting a JSON summary
## Chose
123: overview refuses --json instead of emitting a JSON summary
## Rejected
give overview a --json structured form like context/status
## Because
overview's whole value is human-readable prose plus color; a JSON form would just re-serialize data that status/agents/next already expose machine-readably, so refusing keeps one authoritative machine path per fact instead of two that could drift
