---
id: f-task-424-checkpoint-mutation-is-already-covered-by-task-370
kind: note
note_kind: finding
created: 2026-08-13T15:35:41Z
created_by: a-fixer-sfkfzr
about: "[[424]]"
severity: major
---
# Task 424 checkpoint mutation is already covered by task 370
Commit e836ae3 guards driver.saveState during dry-run at internal/features/orchestration/orchestration.go:526 and TestLoopDryRunLeavesWorkspaceAndGovernorUntouched at internal/features/orchestration/state_test.go:234 snapshots the complete .dacli tree across two previews. Current test passes. Changing the guard condition to false makes the test fail at state_test.go:278 with dry-run 1 modified workspace state, proving restored cycle, governor, and journal writes are caught.
