---
id: f-task-406-branch-lacks-runtime-execution-wiring
kind: note
note_kind: finding
created: 2026-08-13T10:51:30Z
created_by: a-fixer-9e4w6p
about: "[[406]]"
severity: major
---
# Task 406 branch lacks runtime execution wiring
At 376d911, git diff 4f6be10..HEAD changes only internal/providerpolicy, internal/store, and internal/team. internal/features/execution/execution.go still calls execRuntime directly and never imports providerpolicy, loads RuntimeLimits, checks a cooldown before spawn, classifies a nonzero exit, records a breaker, or reports a fallback transition. Thus the policy primitives are unreachable from actual spawn/supervise behavior.
