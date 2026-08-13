---
id: f-task-439-claim-excludes-the-files-required-by-its-regression-and-shared-fix
kind: note
note_kind: finding
created: 2026-08-13T23:10:21Z
created_by: a-fixer-7w7rgg
about: "[[439]]"
severity: major
---
# Task 439 claim excludes the files required by its regression and shared fix
dacli commit refused internal/cli/lifecycle_test.go and internal/gitx/gitx.go as outside the live claim [internal/features/vcs]. The verified fix belongs in gitx.RemoveWorktree so both CLI and loop callers share it, and the end-to-end regression belongs beside internal/cli/lifecycle_test.go; forcing the commit would violate slice ownership.
