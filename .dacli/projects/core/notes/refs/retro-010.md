---
id: r-retro-010
kind: note
note_kind: ref
created: 2026-07-21T22:30:43Z
created_by: a-root
about: [[t-01KY3CGC811EXD56XEN17YTTV3]]
scope: workspace
---
# Retro: 010
## Went well
- adopt reused the whole stack — project, notes, tasks, brief — a repo becomes a dacli workspace in one command

## Didn't go well
- codebase-map headings collided with section parsing (empty in brief); bold labels fixed it — the recurring mdstore-headings-in-content trap

## Improve next time
- any command that writes multi-heading content into a section must use bold labels, not sub-headings; consider a mdstore helper that escapes this

