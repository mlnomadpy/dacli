---
id: f-decision-mirror-dedup-now-preserves-different-structured-payloads
kind: note
note_kind: finding
created: 2026-08-13T22:31:17Z
created_by: a-fixer-4r64qs
about: "[[437]]"
severity: major
---
# Decision mirror dedup now preserves different structured payloads
internal/features/ghmirror/ghmirror.go canonicalNoteFiles now requires normalized Chose/Rejected/Because equivalence before merging near-title decision notes; internal/features/ghmirror/semantic_dedup_test.go reproduces the prior collapse of different decisions.
