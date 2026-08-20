---
id: f-answer-may-task-476-s-claim-expand-to-internal-features-knowledge-knowledge
kind: note
note_kind: finding
created: 2026-08-19T14:36:24Z
created_by: a-root
about: "[[t-01M0CX03CC0N95X4M5ESKRP2E6]]"
---
# Answer: May task 476's claim expand to internal/features/knowledge/knowledge_test.go? Fail-closed versioned overrides require updating its two legacy custom-override fixtures with the declared schema/base header; full go test passes with that two-line fixture change, but dacli commit correctly refused the unclaimed path. Approve that exact test-only expansion so the verified commit can land.
Q (a-maintainer-anf4d3): May task 476's claim expand to internal/features/knowledge/knowledge_test.go? Fail-closed versioned overrides require updating its two legacy custom-override fixtures with the declared schema/base header; full go test passes with that two-line fixture change, but dacli commit correctly refused the unclaimed path. Approve that exact test-only expansion so the verified commit can land.

A: Approved: expand task 476 only to internal/features/knowledge/knowledge_test.go in addition to its existing claims. Restore the two legacy override fixture headers, rerun the focused knowledge and prompt suites, then commit the already-verified implementation. Do not change knowledge production code.
