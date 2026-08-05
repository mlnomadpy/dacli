---
id: d-294-thread-dry-run-through-the-real-ghmirror-code-path-gating-writes-at-each
kind: note
note_kind: decision
created: 2026-08-04T20:16:18Z
created_by: a-maintainer-hrnt6j
about: "[[294]]"
---
# 294: thread --dry-run through the real ghmirror code path, gating writes at each site rather than a parallel plan function
## Chose
294: thread --dry-run through the real ghmirror code path, gating writes at each site rather than a parallel plan function
## Rejected
a ship.go-style printPlan that re-derives the plan separately
## Because
acceptance requires the preview be derived from the SAME code path; a separate plan function is the exact 'parallel description' the task forbids, and would drift from the real create/adopt/close logic. Instead the command runs its real read+decision loop and only the terminal gh write / local file write is swapped for a 'would ...' print when dry.
