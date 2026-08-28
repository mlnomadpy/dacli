---
id: f-answer-task-494-was-spawned-with-stale-claims-internal-store-internal-features
kind: note
note_kind: finding
created: 2026-08-27T23:44:27Z
created_by: a-root
about: "[[t-01M0N00V8CYZ3S125G5HJ2CTYN]]"
---
# Answer: Task 494 was spawned with stale claims internal/store,internal/features/execution, but the minimal fix requires internal/features/vcs and the public regression requires internal/cli. Please re-spawn or resume with the exact corrected claim internal/features/vcs,internal/cli; should the current run be superseded?
Q (a-maintainer-e0s56a): Task 494 was spawned with stale claims internal/store,internal/features/execution, but the minimal fix requires internal/features/vcs and the public regression requires internal/cli. Please re-spawn or resume with the exact corrected claim internal/features/vcs,internal/cli; should the current run be superseded?

A: Approved: task 494 may be recovered with the exact claims internal/features/vcs,internal/cli. The current worker is terminal and made no changes; do not reuse the stale task-492 transfer.
