---
id: d-searchbymarker-replaced-by-a-per-push-markerindex-snapshot-fetched-once-and
kind: note
note_kind: decision
created: 2026-07-22T19:40:41Z
created_by: a-a3xyv593bf
about: [[077]]
---
# searchByMarker replaced by a per-push markerIndex snapshot fetched once and scanned in memory
## Chose
searchByMarker replaced by a per-push markerIndex snapshot fetched once and scanned in memory
## Rejected
keep the full gh issue list per task/decision/finding inside the push loop
## Because
the old path cost one strongly-consistent list call for every unmapped note (O(N) network); one snapshot per push is correct because adoption only ever targets PRIOR-run issues (each note writes its mapping back before the next is searched), so a single list preserves the zero-duplicate crash-recovery guarantee at O(1) network
