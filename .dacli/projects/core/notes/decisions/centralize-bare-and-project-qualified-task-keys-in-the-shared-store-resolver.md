---
id: d-centralize-bare-and-project-qualified-task-keys-in-the-shared-store-resolver
kind: note
note_kind: decision
created: 2026-08-14T00:38:31Z
created_by: a-maintainer-1w0gkw
about: "[[445]]"
---
# Centralize bare and project-qualified task keys in the shared store resolver
## Chose
Centralize bare and project-qualified task keys in the shared store resolver
## Rejected
Teach individual command handlers to strip project qualifiers
## Because
FindTask, TaskIndex, ambiguity guidance, acceptance commands, and orchestration must share one grammar and project boundary; handler parsing would recreate the mismatch.
