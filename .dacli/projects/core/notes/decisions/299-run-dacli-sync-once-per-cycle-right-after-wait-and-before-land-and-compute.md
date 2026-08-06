---
id: d-299-run-dacli-sync-once-per-cycle-right-after-wait-and-before-land-and-compute
kind: note
note_kind: decision
created: 2026-08-06T08:23:48Z
created_by: a-fixer-43t0j0
about: "[[299]]"
---
# 299: run 'dacli sync' once per cycle, right after wait and before LAND, and compute the cycle rollup after LAND (so a task blocked mid-wave via a read-only agent's proposal reports blocked, not merely in-flight)
## Chose
299: run 'dacli sync' once per cycle, right after wait and before LAND, and compute the cycle rollup after LAND (so a task blocked mid-wave via a read-only agent's proposal reports blocked, not merely in-flight)
## Rejected
syncing after the review phase, or not ordering sync relative to LAND at all
## Because
sync's file changes (task status/notes) need to ride in the SAME cycle's record commit recordSelfPR makes, not sit uncommitted for the next cycle to trip over; and the rollup's Blocked bucket is only honest if it reads status AFTER sync applied whatever a read-only build agent could only propose
