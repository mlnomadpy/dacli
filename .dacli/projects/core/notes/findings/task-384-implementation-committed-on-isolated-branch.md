---
id: f-task-384-implementation-committed-on-isolated-branch
kind: note
note_kind: finding
created: 2026-08-12T16:40:23Z
created_by: a-codex-maintainer-s5kkg3
about: "[[384]]"
severity: major
---
# Task 384 implementation committed on isolated branch
Commit 606f104 on branch dacli/384-make-process-dependent-tests-honest-when-process-table-visibility-is-denied adds recorder completion synchronization, an injected-unobservable regression, and explicit process-observation premise skips. Focused execution/procmon suites pass under denied ps; red mutation fails at execruntime_test.go with the unobservable-PID early return. Normal-visibility run, full suite, and lint remain unverified as separately reported.
