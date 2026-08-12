---
id: d-reread-task-check-state-inside-the-existing-per-task-file-lock
kind: note
note_kind: decision
created: 2026-08-12T16:15:30Z
created_by: a-codex-maintainer-xm4nzv
about: "[[364]]"
---
# Reread task-check state inside the existing per-task file lock
## Chose
Reread task-check state inside the existing per-task file lock
## Rejected
Add merge logic to whole-file SaveTask writes
## Because
WithTask already defines the cross-process serialization boundary and preserves unrelated task document fields; command-specific merge logic would be incomplete and would duplicate store semantics.
