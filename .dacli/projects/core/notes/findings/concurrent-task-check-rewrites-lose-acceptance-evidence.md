---
id: f-concurrent-task-check-rewrites-lose-acceptance-evidence
kind: note
note_kind: finding
created: 2026-08-12T16:15:30Z
created_by: a-codex-maintainer-xm4nzv
about: "[[364]]"
severity: major
---
# Concurrent task-check rewrites lose acceptance evidence
Deterministic real-binary regression internal/cli/taskcheck_concurrency_test.go:13 rendezvouses two processes after their initial read. With cmdTaskCheck temporarily restored to its pre-fix direct SaveTask path, go test ./internal/cli -run TestTaskCheckConcurrentProcessesPreserveDifferentCriteria -count=1 fails at taskcheck_concurrency_test.go:56: persisted acceptance = [1/2], want [2/2].
