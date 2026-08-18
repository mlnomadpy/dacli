---
id: f-task-465-claim-excludes-required-regression-and-advisory-snapshot-files
kind: note
note_kind: finding
created: 2026-08-18T15:14:49Z
created_by: a-maintainer-dgyp5f
about: "[[t-01M0AEG5F23TRH6BAR9HT38ZP1]]"
severity: major
---
# Task 465 claim excludes required regression and advisory-snapshot files
dacli commit refused with exit 3: claim permits journal.go, orchestration.go, and internal/store, but the required fault/restart tests modify driver_test.go, journal_test.go, landing_policy_test.go, state_test.go, and explicit advisory snapshot injection modifies state.go. Per claim policy the refusal was not retried or forced. The verified implementation remains uncommitted in this isolated worktree; widen the task claim to internal/features/orchestration (or at minimum those five paths) and rerun dacli commit/push/pr.
