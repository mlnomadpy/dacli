---
id: d-next-and-critical-path-both-exclude-blocked-tasks-from-the-schedule
kind: note
note_kind: decision
created: 2026-07-22T16:25:53Z
created_by: a-8g6b17xcdq
about: [[051]]
---
# next and critical-path both EXCLUDE blocked tasks from the schedule
## Chose
next and critical-path both EXCLUDE blocked tasks from the schedule
## Rejected
both include blocked as zero-duration nodes with an in-output flag
## Because
cmdNext already excluded blocked; making critical-path match (rather than teaching both to carry blocked nodes) is the smaller, consistent change. Edges now point only to scheduled nodes, so a blocked dependency can never trigger ComputeCPM's 'edge references unknown task'; readiness against a blocked dep is still enforced by ready().
