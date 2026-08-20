---
id: d-treat-brief-as-an-i-o-assembly-service-and-document-serialized-state-explicitly
kind: note
note_kind: decision
created: 2026-08-19T14:41:59Z
created_by: a-fixer-wcmm5q
about: "[[t-01M0AEG5QK6WDSJQBZTG7Z8JWW]]"
github:
  issue: 753
  repo: mlnomadpy/dacli
---
# Treat brief as an I/O assembly service and document serialized state explicitly
## Chose
Treat brief as an I/O assembly service and document serialized state explicitly
## Rejected
Preserve the pure-L4 and lock-free descriptions
## Because
brief.Assemble reads workspace state directly and store/features serialize shared transitions with file locks
