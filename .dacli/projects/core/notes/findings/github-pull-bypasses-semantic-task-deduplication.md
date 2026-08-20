---
id: f-github-pull-bypasses-semantic-task-deduplication
kind: note
note_kind: finding
created: 2026-08-19T12:14:46Z
created_by: a-maintainer-ebqr9f
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# GitHub pull bypasses semantic task deduplication
The pull path skips mapped issue numbers, dacli markers, and closed issues around internal/features/ghmirror/ghmirror.go:900, but does not run task add near-duplicate detection. The playbook must not imply that pull then task-list deduplication prevents already-created duplicates.
