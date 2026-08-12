---
id: f-loop-spawns-inherited-the-fixed-five-minute-spawn-default
kind: note
note_kind: finding
created: 2026-08-12T15:38:24Z
created_by: a-codex-maintainer-vxzmpg
about: "[[378]]"
severity: major
---
# Loop spawns inherited the fixed five-minute spawn default
internal/features/orchestration/orchestration.go build and review spawn argument construction omitted --timeout, so execution's default recorded timeout_s: 300 regardless of task Te. Exact driver regression restores 300 and fails TestLoopDerivesWorkerTimeoutFromEachTaskEstimate: got --timeout 300, want 1800.
