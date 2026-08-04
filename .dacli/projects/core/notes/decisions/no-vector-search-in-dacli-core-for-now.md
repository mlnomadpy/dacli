---
id: d-no-vector-search-in-dacli-core-for-now
kind: note
note_kind: decision
created: 2026-08-04T09:37:13Z
created_by: a-root
---
# no vector search in dacli core for now
## Chose
no vector search in dacli core for now
## Rejected
embedding-based search over roles, skills, notes and tasks
## Because
at 18 roles / 1 skill / 413 notes the read path is already 25-43ms so there is no retrieval problem; it would cost the zero-dependency property; embedding similarity is non-deterministic across model versions while dacli's value is a trustworthy record; and the existing fuzzy matcher is already too loose rather than too tight. Revisit only for near-duplicate detection, and then as an optional external backend.
