---
id: f-task-add-ignored-problem-and-acceptance-text-during-duplicate-checks
kind: note
note_kind: finding
created: 2026-08-16T17:45:45Z
created_by: a-maintainer-n5gm5y
about: "[[t-01KZZR4CN0HWN232ZD2GYGQDFP]]"
severity: major
---
# Task add ignored problem and acceptance text during duplicate checks
internal/features/planning/planning.go passed only title into internal/store/similarity.go, so differently titled tasks with the same reproduction and acceptance boundary bypassed the local refusal. The new structured input is evaluated before CreateTask, preserving non-mutation on exit 3.
