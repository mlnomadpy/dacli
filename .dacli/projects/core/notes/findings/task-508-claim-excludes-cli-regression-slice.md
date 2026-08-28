---
id: f-task-508-claim-excludes-cli-regression-slice
kind: note
note_kind: finding
created: 2026-08-28T00:21:25Z
created_by: a-maintainer-z0nyk9
about: "[[t-01M1068M8HJ9G8XCXMEMVE2V8D]]"
severity: moderate
---
# Task 508 claim excludes CLI regression slice
dacli commit refused internal/features/planning/dependency_test.go because run 01M12VJA2442GSFSYN7BS6C4QE claims only internal/store. The store boundary used by both direct task depend and proposal sync has the multi-project persistence regression; a command-handler-level duplicate could not be committed under this claim.
