---
id: f-task-431-github-issue-and-mirror-preview-are-network-blocked
kind: note
note_kind: finding
created: 2026-08-13T20:04:03Z
created_by: a-codex-maintainer-tt3db3
about: "[[431]]"
severity: moderate
---
# Task 431 GitHub issue and mirror preview are network-blocked
gh issue view 605 and the required dacli github push core 431 --dry-run both failed because api.github.com was unreachable; no public mutation was attempted.
