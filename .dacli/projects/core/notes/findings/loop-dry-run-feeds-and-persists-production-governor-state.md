---
id: f-loop-dry-run-feeds-and-persists-production-governor-state
kind: note
note_kind: finding
created: 2026-08-12T15:43:48Z
created_by: a-codex-maintainer-3vy9w1
about: "[[370]]"
severity: major
---
# Loop dry-run feeds and persists production governor state
Focused regression TestLoopDryRunLeavesWorkspaceAndGovernorUntouched starts at zero_streak 2/3 and fails: preview prints 'thrash guard tripped'; loop() also calls saveState at checkpoints, rewriting cycle journal, loop status, and governor snapshot (internal/features/orchestration/orchestration.go:599,674).
