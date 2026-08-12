---
id: f-task-371-claim-retained-only-documentation-and-omitted-three-implementation
kind: note
note_kind: finding
created: 2026-08-12T16:48:08Z
created_by: a-codex-maintainer-nmzkpw
about: "[[385]]"
severity: major
---
# Task 371 claim retained only documentation and omitted three implementation trees
Reproduced with the completed task 371 acceptance text: internal/store/claimhints_test.go:15 receives ClaimHints=[docs/RUNTIMES.md], omitting internal/store, internal/features/execution, and internal/cli although commit 56be159 changed six files in those trees.
