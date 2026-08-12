---
id: d-wait-for-a-recorder-completion-marker-before-reading-detached-captures
kind: note
note_kind: decision
created: 2026-08-12T16:38:18Z
created_by: a-codex-maintainer-s5kkg3
about: "[[384]]"
---
# Wait for a recorder completion marker before reading detached captures
## Chose
Wait for a recorder completion marker before reading detached captures
## Rejected
Treat an unreadable or absent ProcState result as detached-child exit
## Because
Process-table denial and process exit produce the same ProcState result, while a marker written after stdin capture is an observable completion signal and preserves the exact 164000-byte assertion
