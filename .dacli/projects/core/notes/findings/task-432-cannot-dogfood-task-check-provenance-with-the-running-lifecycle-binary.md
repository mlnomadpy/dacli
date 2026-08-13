---
id: f-task-432-cannot-dogfood-task-check-provenance-with-the-running-lifecycle-binary
kind: note
note_kind: finding
created: 2026-08-13T21:31:18Z
created_by: a-fixer-dq7v5k
about: "[[432]]"
severity: minor
---
# Task 432 cannot dogfood task-check provenance with the running lifecycle binary
The required /private/tmp/dacli-loop-current predates commit e85b5a4: task check 432 --n 1 --verify ... returned usage exit 2 with unknown flag --verify. Acceptance was therefore verified in code tests, while lifecycle check events must use the binary's existing syntax until this branch lands.
