---
id: f-task-399-github-mirror-preview-is-blocked-by-sandbox-networking
kind: note
note_kind: finding
created: 2026-08-12T20:13:28Z
created_by: a-codex-maintainer-csf6ta
about: "[[399]]"
severity: major
---
# Task 399 GitHub mirror preview is blocked by sandbox networking
After commit 376d729, required dacli github push core 399 --dry-run failed because gh could not connect to api.github.com. No public mirror mutation was attempted.
