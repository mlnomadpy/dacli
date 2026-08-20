---
id: f-task-471-remains-blocked-by-out-of-claim-capability-and-trust-invariants
kind: note
note_kind: finding
created: 2026-08-19T13:36:43Z
created_by: a-maintainer-d7rbr1
about: "[[t-01M0AGCX29Q047FZHKNG3YV0WC]]"
severity: major
---
# Task 471 remains blocked by out-of-claim capability and trust invariants
Verified on commit b7d3101 with GOCACHE=/tmp/dacli-go-cache-471 go test ./...: internal/cli TestEveryCommandDeclaresItsCapability requires worktree reclaim in internal/cli/capability_invariant_test.go:139, and TestTrustDocListsEveryMutatingCommand requires docs/TRUST.md to name it. This run inherits the task's internal/features/vcs,internal/gitx claim, so those edits cannot be governed-committed without an explicit claim expansion. Focused go test ./internal/features/vcs ./internal/cli (before the invariant aggregation completed) and go test -race ./internal/features/vcs confirmed the vcs behavior itself; the full suite is red and no acceptance box was marked.
