---
id: 01KY61GVQZNZ5N4CKQC5R75XD9
kind: event
event_kind: finding
created: 2026-07-22T23:10:39Z
created_by: a-7ahpwv8p4f
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
loop Idle path is ungoverned: unbounded review-spawns, thrash-halt bypassed, non-yolo never checkpoints on empty backlog

internal/features/orchestration/orchestration.go loop(): the Idle branch (174-183) calls reviewPhase()+sleep(gov.Idle)+continue on every empty-backlog scan but NEVER calls gov.AfterCycle. NoProgressHalt (governor.go:113) trips ONLY inside AfterCycle, reached only from the runCycle path (orchestration.go:186-188). So there is no consecutive-idle cap: an empty backlog the reviewer cannot refill (a realistic terminal state, e.g. all findings already fixed) makes the loop spin forever, spawning a BLOCKING review agent (reviewPhase -> run 'review' 'spawn', line 273, no --detach; real token cost) every gov.Idle interval (default 30m, :107) with no halt. Two contract violations: (1) the log at :153 prints 'thrash-halt after %d idle cycles' using NoProgressHalt, but idle iterations never advance zeroStreak, so the guard is OFF on exactly the path it names; (2) non-yolo mode is documented as run-one-cycle-then-checkpoint (:197-200 returns nil), but the Idle branch has no yolo/return guard, so an empty backlog in non-yolo silently becomes a perpetual idle loop that never checkpoints or halts. Impact: the governor's core promise (bound a perpetual machine) is bypassed whenever the backlog is empty -> unbounded review-agent spend. This is a distinct axis from task 085 (which addresses landed accuracy in EXECUTED cycles under --pr --auto); the Idle path is never governed at all. Fix: bound consecutive idles (route idle through AfterCycle(landed=0) or a dedicated idle-streak cap), honor checkpoint/yolo semantics on the idle path, and correct the :153 wording.
