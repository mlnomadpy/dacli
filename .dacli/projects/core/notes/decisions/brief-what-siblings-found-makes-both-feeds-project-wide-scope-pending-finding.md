---
id: d-brief-what-siblings-found-makes-both-feeds-project-wide-scope-pending-finding
kind: note
note_kind: decision
created: 2026-07-22T19:26:27Z
created_by: a-5gg79k0bcq
about: [[074]]
---
# brief 'What siblings found' makes BOTH feeds project-wide: scope pending finding events to the project, matching notes
## Chose
brief 'What siblings found' makes BOTH feeds project-wide: scope pending finding events to the project, matching notes
## Rejected
make finding notes task-scoped so both feeds are task-scoped
## Because
the section is titled 'What siblings found' and its comment says findings are 'visible tree-wide the instant written'; materialized NOTES were already project-wide (store.ListNotes per-project) and that cross-sibling behavior is demonstrated in real briefs, so scoping events UP to project (not scoping notes DOWN to task) preserves the intended cross-task sibling learning and removes the sync-boundary visibility flip (issue #21)
