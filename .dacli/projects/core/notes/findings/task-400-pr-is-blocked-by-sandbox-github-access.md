---
id: f-task-400-pr-is-blocked-by-sandbox-github-access
kind: note
note_kind: finding
created: 2026-08-12T20:24:04Z
created_by: a-codex-maintainer-w6vv23
about: "[[400]]"
severity: major
---
# Task 400 PR is blocked by sandbox GitHub access
The required dacli github push core 400 --dry-run failed because gh could not connect to api.github.com. No public mirror mutation, branch push, or PR creation was attempted. From a network-enabled context rerun the dry-run, then dacli push --task 400 and dacli pr --task 400 --with-verdicts --auto.
