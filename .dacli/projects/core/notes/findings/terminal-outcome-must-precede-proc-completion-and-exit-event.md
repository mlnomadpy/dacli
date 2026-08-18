---
id: f-terminal-outcome-must-precede-proc-completion-and-exit-event
kind: note
note_kind: finding
created: 2026-08-18T14:20:55Z
created_by: a-maintainer-zppm9n
about: "[[t-01M0AEG5AQPVJTH41MJNFRGSSX]]"
severity: major
---
# Terminal outcome must precede proc completion and exit event
internal/features/execution/execution.go finalizeRun previously completed proc.txt before writing outcome.md and ignored both errors; a failed outcome write could release claims and append a successful agent-exit event with no terminal artifact. finalizeRunChecked now atomically writes outcome first and returns before proc completion/event append on failure.
