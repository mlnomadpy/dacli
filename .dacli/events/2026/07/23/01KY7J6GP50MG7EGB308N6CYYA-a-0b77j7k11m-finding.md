---
id: 01KY7J6GP50MG7EGB308N6CYYA
kind: event
event_kind: finding
created: 2026-07-23T13:21:20Z
created_by: a-0b77j7k11m
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
loop idle-cycle review spawns are never charged to the token window, defeating --window-tokens in the loop's steady state

In internal/features/orchestration/orchestration.go the token window is only ever advanced from AfterCycle (governor.go:142-144, g.windowSpent += tokens), which is fed exclusively by runCycle's deferred RunsTokensSince charge (orchestration.go:359-361). But the Idle branch of loop() (orchestration.go:304-313) does NOT go through runCycle or AfterCycle: it calls d.reviewPhase() — which spawns a REAL token-spending reviewer (reviewPhase orchestration.go:486-494 -> 'spawn --task <ref> --role <reviewRole>') — then d.sleep(Idle) and 'continue's straight back to the top of the loop. So every idle tick spends reviewer tokens that are never summed into windowSpent. Consequence: the loop's steady state with an empty/quiet backlog IS the Idle path (each idle review regenerates work — the self-feeding design, orchestration.go:307), so the dominant token cost of a long-running loop is exactly the cost the --window-tokens guard cannot see. windowSpent stays flat across idle ticks, so Before()'s window check (governor.go:129-131, windowSpent >= WindowTokens) never trips and the loop keeps spawning reviewers regardless of the configured budget. 'dacli loop status' also under-reports: saveState writes WindowTokens: d.gov.WindowSpent() (orchestration.go:263), which does not grow during idle. Task 091 (done) wired per-CYCLE accounting into runCycle only ('runCycle sums the cycle's spawn usage.txt token actuals') and left the idle path uncounted — this is the gap it did not close. Secondary (same root): the idle review spawn also ignores --max-tokens (perCycleTok is only appended to BUILD spawns at orchestration.go:376-378, never to the reviewPhase spawn), so it is unbounded per-run as well as uncharged.
