---
id: f-task-370-implementation-committed-on-isolated-branch
kind: note
note_kind: finding
created: 2026-08-12T15:48:37Z
created_by: a-codex-maintainer-3vy9w1
about: "[[370]]"
severity: major
---
# Task 370 implementation committed on isolated branch
Commit e836ae3 on dacli/370-make-loop-dry-run-side-effect-free makes preview checkpoints and direct mutators inert, exits before governor accounting, and adds a two-run whole-workspace digest regression at streak 2/3 while asserting all planned phases remain visible. Focused orchestration tests, gofmt, and go vet pass; full race blockers and missing lint binary are reported separately.
