---
id: f-task-408-explicit-new-boundary-now-outranks-generic-execution-prose
kind: note
note_kind: finding
created: 2026-08-13T14:58:53Z
created_by: a-fixer-dd0fvf
about: "[[419]]"
severity: major
---
# Task 408 explicit new boundary now outranks generic execution prose
Mutation evidence: before the fix, GOCACHE=/private/tmp/dacli-419-gocache go test ./internal/store -run TestClaimHintsPreservesExplicitNewTask408Boundary failed at internal/store/claimhints_test.go:259 with ClaimHints = [internal/features/execution], missing contracts/controlplane/v1. The fixed ClaimHints accepts a new descendant under an existing top-level boundary and suppresses weaker vocabulary inference.
