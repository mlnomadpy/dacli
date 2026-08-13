---
id: f-task-422-already-added-the-terminal-state-task-436-needs-but-lacks-composition
kind: note
note_kind: finding
created: 2026-08-13T21:03:10Z
created_by: a-codex-maintainer-8r5s5s
about: "[[436]]"
severity: major
---
# Task 422 already added the terminal state task 436 needs but lacks composition coverage
internal/features/execution/execution.go:2790 recovers stale claimed runs with procmon.CompleteRecord and runLifecycleLive at execution.go:2850 treats rec.Outcome as terminal before startup/transcript grace. No test currently calls cmdAgents before cmdWait for multiple recovered named runs.
