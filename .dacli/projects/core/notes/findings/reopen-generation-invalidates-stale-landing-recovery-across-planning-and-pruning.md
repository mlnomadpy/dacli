---
id: f-reopen-generation-invalidates-stale-landing-recovery-across-planning-and-pruning
kind: note
note_kind: finding
created: 2026-08-17T15:47:44Z
created_by: a-maintainer-q8e6rb
about: "[[t-01M05Y78XTYNQ4GP3AV396JFD6]]"
severity: major
---
# Reopen generation invalidates stale landing recovery across planning and pruning
Verified in internal/features/orchestration/driver_test.go:847 and internal/store/worktrees_test.go:22: public ReopenTask increments generation, legacy/current pending_accept entries from an earlier generation are discarded before verifier recovery, the must task returns to ready selection, and a clean checkout on the previously merged branch is not reclaimable. Mutation proofs failed at driver_test.go:890 when reconciliation was disabled and worktrees_test.go:92 when the pruning guard was removed.
