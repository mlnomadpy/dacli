---
id: d-pull-idempotency-keyed-on-the-local-task-s-github-mapping-issue-number-not-a
kind: note
note_kind: decision
created: 2026-07-22T17:35:09Z
created_by: a-sdpxn53045
about: [[057]]
---
# Pull idempotency keyed on the local task's github: mapping (issue number), not a body marker
## Chose
Pull idempotency keyed on the local task's github: mapping (issue number), not a body marker
## Rejected
Edit the imported issue to stamp a dacli marker into its body so the next pull skips it by marker
## Because
Pull is read-only against the remote (it never edits an issue), so an adopted issue's body never gains a marker. Writing the github: issue/repo block back onto the new task and skipping any issue already in that mapped-number set gives idempotency without touching GitHub — a re-pull re-lists the same issue, sees it mapped, and skips. shouldImport() encodes both skips (dacli-authored body marker OR already-mapped).
