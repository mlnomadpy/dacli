---
id: f-task-removal-has-two-distinct-live-agent-safety-boundaries
kind: note
note_kind: finding
created: 2026-08-14T09:07:23Z
created_by: a-maintainer-3gxynh
about: "[[t-01KZZR4CR10XX2BAZG1Y1ZDDZ7]]"
severity: moderate
---
# Task removal has two distinct live-agent safety boundaries
internal/store/remove.go liveClaimants refuses only when a live run points at the specific task; root cross-owner authorization additionally must inspect whether the task owner has any live run, even one whose run record currently names another task. Regression added in internal/features/planning/reopen_test.go.
