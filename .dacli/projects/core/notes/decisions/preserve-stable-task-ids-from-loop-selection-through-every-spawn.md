---
id: d-preserve-stable-task-ids-from-loop-selection-through-every-spawn
kind: note
note_kind: decision
created: 2026-08-13T15:54:37Z
created_by: a-fixer-f6typj
about: "[[423]]"
---
# Preserve stable task IDs from loop selection through every spawn
## Chose
Preserve stable task IDs from loop selection through every spawn
## Rejected
Construct project-qualified sequence references at each call site
## Because
Selected store.Task values already carry globally stable IDs; forwarding them avoids reconstructing identity, covers implementation, estimator, and review spawns uniformly, and leaves operator-entered short refs unchanged.
