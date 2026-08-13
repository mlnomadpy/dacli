---
id: f-loop-routing-now-persists-explanation-and-calibrates-first-pass-outcomes
kind: note
note_kind: finding
created: 2026-08-13T14:32:30Z
created_by: a-fixer-ngpzz6
about: "[[407]]"
severity: major
---
# Loop routing now persists explanation and calibrates first-pass outcomes
internal/team/routing.go applies hard eligibility before provider-neutral measured ranking; internal/store/calibration.go derives first-attempt success with sample counts; internal/features/orchestration/orchestration.go writes .dacli/loop/routing/cycle-N-task-N.json and restricts paused-runtime alternatives to fallback_to. Mutation evidence: removing writeRoutingExplanation made TestLoopRoutesEachTaskToCheapestCapableRoleByTe fail with 'loop did not record routing explanation: ... no such file or directory'.
