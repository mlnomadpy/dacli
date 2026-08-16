---
id: d-interpret-only-terminal-recursive-tree-claim-syntax
kind: note
note_kind: decision
created: 2026-08-16T18:36:18Z
created_by: a-maintainer-w9qqkt
about: "[[t-01KZYQ5E9PFVWRVMSWPB39E38K]]"
---
# Interpret only terminal recursive-tree claim syntax
## Chose
Interpret only terminal recursive-tree claim syntax
## Rejected
Apply general filepath glob matching to all wildcard-bearing claims
## Because
A trailing /** is the documented recursive claim form and reduces to the existing literal-directory containment rule; interpreting embedded or single-star patterns would silently authorize and reserve paths not previously covered.
