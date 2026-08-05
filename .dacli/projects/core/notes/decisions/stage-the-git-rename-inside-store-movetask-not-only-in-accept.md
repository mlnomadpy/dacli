---
id: d-stage-the-git-rename-inside-store-movetask-not-only-in-accept
kind: note
note_kind: decision
created: 2026-08-04T20:11:25Z
created_by: a-maintainer-yd9g1p
about: "[[273]]"
---
# Stage the git rename inside store.MoveTask, not only in accept
## Chose
Stage the git rename inside store.MoveTask, not only in accept
## Rejected
Stage only inside the acceptance slice's acceptOne/acceptAll
## Because
MoveTask is the single canonical status-mover (task done, integrate, sync, planning, collab all route through it); staging the delete+add there closes the uncommitted-move->git-resurrects->duplicate-task-file class for EVERY close path, not just accept. store already imports gitx, so no new slice dependency. Best-effort so non-git unit workspaces are unaffected.
