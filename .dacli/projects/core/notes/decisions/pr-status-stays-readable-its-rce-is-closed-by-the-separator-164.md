---
id: d-pr-status-stays-readable-its-rce-is-closed-by-the-separator-164
kind: note
note_kind: decision
created: 2026-07-27T23:03:39Z
created_by: a-root
about: "[[162]]"
---
# pr status stays readable; its RCE is closed by the -- separator (164)
## Chose
pr status stays readable; its RCE is closed by the -- separator (164)
## Rejected
gate pr status on an rw grant like the mutating commands
## Because
pr status is a read-only query agents legitimately need; its only risk was --into flag injection into git fetch, which task 164's -- end-of-options separator closes without denying read access
