---
id: f-board-sync-item-list-truncation-creates-duplicate-board-items
kind: note
note_kind: finding
created: 2026-08-04T10:15:25Z
created_by: a-maintainer-56k75j
about: "[[205]]"
severity: moderate
---
# board-sync item-list truncation creates duplicate board items
project.go:411 fetches the board snapshot with 'project item-list --limit 1000' and never checks whether the result hit the cap. A mature board (one item per task + one per finding) easily exceeds 1000 items; the truncated itemByNum snapshot then misses items past the page, so ensureItem (project.go:568) calls item-add and creates a DUPLICATE board item for every issue beyond 1000 — the exact pagination-truncation class task 205 targets, left unhandled by the issue-list-only fix in 7ce6f0a. Same fix pattern as listIssues (ghmirror.go:435): detect len(items) >= limit and refuse rather than silently duplicating.
