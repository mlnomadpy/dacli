---
id: d-contrib-dedups-by-skipping-applied-eventfindings-count-the-event-only-while
kind: note
note_kind: decision
created: 2026-07-22T19:19:55Z
created_by: a-n2nn0v2g31
about: [[073]]
---
# contrib dedups by skipping APPLIED EventFindings (count the event only while unsynced)
## Chose
contrib dedups by skipping APPLIED EventFindings (count the event only while unsynced)
## Rejected
count NoteFindings only and drop the event-against count entirely
## Because
A ro reviewer's finding-against exists first as a pending EventFinding (no note yet) and only becomes a NoteFinding after the owner runs dacli sync. Counting notes only would UNDER-count every ro finding until it is synced. Gating the event count on !e.Applied counts each finding exactly once in every state: pending event -> counted as event; applied event -> skipped, its synced note counted; rw direct note -> counted as note. eventlog.Event already exposes Applied, so no new I/O.
