---
id: d-143-opt-in-flags-reject-known-helper-adopted-on-task-add-note-add-task-check
kind: note
note_kind: decision
created: 2026-07-26T20:46:06Z
created_by: a-x7vfysmvvg
about: [[143]]
---
# 143: opt-in Flags.Reject(known...) helper adopted on task add/note add/task check
## Chose
143: opt-in Flags.Reject(known...) helper adopted on task add/note add/task check
## Rejected
a global unknown-flag reject inside ParseFlags
## Because
ParseFlags is shared by run, which forwards unknown flags via Raw() by design; a global reject would break that pass-through. Reject is opt-in per-command so agent-facing mutating commands (task add, note add, task check) get exit-2 on a typo'd flag while run stays untouched.
