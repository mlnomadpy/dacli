---
id: d-use-disposable-loaded-views-and-invalidatable-task-phase-snapshots
kind: note
note_kind: decision
created: 2026-08-20T08:08:13Z
created_by: a-maintainer-p5kmb7
about: "[[t-01M0AEG5K7JF96HV0RJ5K17NJN]]"
github:
  issue: 756
  repo: mlnomadpy/dacli
---
# Use disposable loaded views and invalidatable task phase snapshots
## Chose
Use disposable loaded views and invalidatable task phase snapshots
## Rejected
Add a process-long cache with implicit filesystem freshness
## Because
Commands and loop phases need repeatable reads, while sibling writes and task transitions must become visible only at explicit reload boundaries and stale task snapshots must refuse use after invalidation.
