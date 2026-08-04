---
id: d-extended-205-pagination-hit-limit-detection-to-the-board-sync-item-list
kind: note
note_kind: decision
created: 2026-08-04T10:18:05Z
created_by: a-maintainer-56k75j
about: "[[205]]"
---
# Extended 205 pagination-hit-limit detection to the board-sync item-list, refusing on a truncated snapshot
## Chose
Extended 205 pagination-hit-limit detection to the board-sync item-list, refusing on a truncated snapshot
## Rejected
Leaving item-list unguarded because the prior commit only covered gh issue list, or silently paginating/raising the limit
## Because
The prior 205 commit (7ce6f0a) fixed the issue-list surface but left project.go:411 'project item-list --limit 1000' unchecked — the identical pagination-truncation class. A mature board carries one item per task plus one per finding and readily exceeds 1000; a truncated itemByNum snapshot makes ensureItem item-add a DUPLICATE board item for every issue past the page, corrupting the board the sync exists to keep idempotent. Refusing (mirroring listIssues at ghmirror.go:435) upholds axiom 5 'never let the record lie' rather than silently duplicating; auto-paginating would hide the operational signal that the board has outgrown a single page.
