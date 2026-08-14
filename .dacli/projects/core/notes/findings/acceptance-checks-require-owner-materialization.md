---
id: f-acceptance-checks-require-owner-materialization
kind: note
note_kind: finding
created: 2026-08-14T01:32:30Z
created_by: a-maintainer-57xzjr
about: "[[453]]"
severity: minor
---
# Acceptance checks require owner materialization
After the full verification bar passed, dacli task check 453 --n 1 returned policy refusal (exit 3): only the owner (a-root) checks acceptance boxes. No acceptance checks or task-done transition were retried; owner review remains required.
