---
id: f-reproduced-three-individually-correct-guards-deadlock-the-agent-close-path
kind: note
note_kind: finding
created: 2026-08-10T13:18:00Z
created_by: a-root
about: "[[312]]"
severity: major
origin: internal/features/planning/planning.go, internal/eventlog/sync.go
---
# Reproduced: three individually-correct guards deadlock the agent close path
Found by RUNNING the loop with a mock runtime in a no-remote workspace, not by reading code — no amount of review had surfaced it. Exact sequence, verified as a real rw spawned agent: (1) 'dacli task claim 003' -> 'claim recorded as event (not the owner); the owner applies it on sync', so the agent is NOT the owner for its own next command; (2) 'dacli task check 003 --all' -> REFUSED, 'only the owner (loop) checks acceptance boxes; report a finding instead'; (3) 'dacli task done 003' -> 'done proposed as event (not the owner)'; (4) 'dacli sync' applies the claim but leaves propose:done pending, because 284's guard correctly refuses a close whose acceptance boxes are unchecked. The agent therefore cannot ever satisfy the condition its own close requires. 'accept --all' does not help: it consumes accept-propose COMMENT events, a different channel from the propose-status event 'task done' files, and reports 'no tasks proposed for acceptance'. Observed consequence in a real cycle: both tasks committed real work on their branches (1 commit ahead of main each), stayed open, and the loop halted on 'no net progress for 3 consecutive cycles' with the work finished but unlandable. This is the live remainder of issue #382 item 1 — the prLandStatus half is fixed (local merges now count as landings), but the CLOSE half deadlocks before landing is ever considered. Each guard is right on its own; the composition is what fails.
