---
id: f-github-pull-bypassed-the-shared-live-task-duplicate-policy
kind: note
note_kind: finding
created: 2026-08-19T13:29:33Z
created_by: a-maintainer-ycbqam
about: "[[t-01M0CZAN6QKQC26961BMZMF79N]]"
severity: major
---
# GitHub pull bypassed the shared live-task duplicate policy
Confirmed internal/features/ghmirror/ghmirror.go pull previously called shouldImport and then CreateTask directly, while internal/features/planning/planning.go used internal/store/similarity.go; a near-duplicate inbound issue could therefore create a second open task.
