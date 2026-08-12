---
id: f-task-381-claim-excludes-required-validator-and-command-regression-paths
kind: note
note_kind: finding
created: 2026-08-12T16:59:57Z
created_by: a-codex-maintainer-gqkrc4
about: "[[381]]"
severity: major
---
# Task 381 claim excludes required validator and command regression paths
dacli commit refused with exit 3 because the task claim is only [internal/store], excluding internal/spm, internal/features/planning, internal/features/insight, and internal/cli. Those paths are explicitly required by acceptance (shared ThreePoint validation, creation/resizing command regressions, and critical-path output). The complete passing focused implementation remains uncommitted on branch dacli/381-reject-non-finite-task-estimates-before-they-corrupt-scheduling-and-timeout; per claim policy I did not retry with --force or commit a partial store-only fix.
