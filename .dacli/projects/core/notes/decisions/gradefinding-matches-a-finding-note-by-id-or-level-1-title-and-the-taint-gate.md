---
id: d-gradefinding-matches-a-finding-note-by-id-or-level-1-title-and-the-taint-gate
kind: note
note_kind: decision
created: 2026-07-22T13:52:49Z
created_by: a-9v3qa4cstk
about: [[029]]
---
# GradeFinding matches a finding note by id OR level-1 title, and the taint gate reuses a shared externalRadius helper
## Chose
GradeFinding matches a finding note by id OR level-1 title, and the taint gate reuses a shared externalRadius helper
## Rejected
GradeFinding keyed on note id only; and a separate taint computation inside the gate distinct from --advise
## Because
verify identifies the judged finding by its CLAIM TEXT (latestFinding/--claim), which is the note's level-1 title, not its id — keying on id alone would never match the thing verify actually graded, so GradeFinding accepts either. And factoring store.Taint('external:')/ExposedBriefs into one externalRadius(w,t) helper in execution.go means --advise (display) and the spawn gate (refusal) compute identical blast radius from one place, so they can never disagree — a duplicated computation could drift and warn-but-not-block (or vice versa). Both stay within the STRICT scope (execution.go only; no cross-slice import).
