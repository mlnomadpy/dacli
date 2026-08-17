---
id: f-task-460-claim-excludes-required-orchestration-and-planning-files
kind: note
note_kind: finding
created: 2026-08-17T15:40:08Z
created_by: a-maintainer-z795wm
about: "[[t-01M05Y78XTYNQ4GP3AV396JFD6]]"
severity: major
---
# Task 460 claim excludes required orchestration and planning files
dacli commit refused exit 3 because the live claim is only [internal/store], but durable pending-accept invalidation requires internal/features/orchestration/{orchestration.go,journal.go,tests} and the public reopen regression requires internal/features/planning/reopen_test.go. The implementation and full verification passed locally, but committing with --force would violate claim isolation. Manual recovery: respawn or widen the task claim to include internal/store, internal/features/orchestration, and internal/features/planning.
