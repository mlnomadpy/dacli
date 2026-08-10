---
id: 01KZ68GQ06HVCD10F5H1W6N6BY
kind: event
event_kind: commit
created: 2026-08-04T11:28:36Z
created_by: a-root
origin: agent
applied: true
---
a847615 record the wave: retro, the ship-test CI flake (256), and this session's agent roster

256 — TestShipRecordMessageReportsActualMerges failed CI on a PR that
touches no Go code, and the failure is in t.TempDir cleanup
('directory not empty'), not the assertion. Sixteen local runs are
clean, so it is a teardown race, not a defect in the change under test.
Filed rather than re-run and forgotten.

The retro is the honest version: invariant tests kept finding the bug
class this codebase actually produces, worktree spawns landed six
independent fixes without collisions, and the integrator earned its keep
by REFUSING to merge and filing the reason. Against that: I merged by
hand instead of spawning the role that does it, the seq allocator handed
out 250 three times, and team assign routed everything to a role whose
runtime cannot write.
role: root
