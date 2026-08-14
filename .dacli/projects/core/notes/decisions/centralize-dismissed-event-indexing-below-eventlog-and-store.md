---
id: d-centralize-dismissed-event-indexing-below-eventlog-and-store
kind: note
note_kind: decision
created: 2026-08-14T09:30:48Z
created_by: a-maintainer-rmzh0s
about: "[[t-01KZZSD1K4YT88J0YYB5ZPD75R]]"
---
# Centralize dismissed-event indexing below eventlog and store
## Chose
Centralize dismissed-event indexing below eventlog and store
## Rejected
Duplicate dismissal filtering in eventlog.List and store.RemoveTask
## Because
eventlog.Sync already imports store, so a lower dependency-neutral predicate avoids an import cycle while ensuring pending reads and canonical reference checks use the same terminal disposition set
