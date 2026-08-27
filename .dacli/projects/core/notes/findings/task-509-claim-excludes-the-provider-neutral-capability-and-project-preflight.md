---
id: f-task-509-claim-excludes-the-provider-neutral-capability-and-project-preflight
kind: note
note_kind: finding
created: 2026-08-27T22:51:48Z
created_by: a-maintainer-ptwdk2
about: "[[t-01M1068MEG379NZ2SE5EH6DYZC]]"
severity: major
---
# Task 509 claim excludes the provider-neutral capability and project-preflight paths
dacli commit refused docs/RUNTIMES.md, internal/features/orchestration/profile.go, internal/features/orchestration/profile_test.go, internal/store/runtimefiles.go, and internal/store/runtimefiles_test.go because the live claim is only internal/features/execution and internal/cli. A correct slice-isolated implementation cannot put shared capability vocabulary or operating-profile gating inside execution. Expand the claim to internal/store, internal/features/orchestration, and docs/RUNTIMES.md, then resume the existing tested worktree; do not use --force.
