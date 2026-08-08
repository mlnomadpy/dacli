---
id: d-filed-the-idle-cycle-token-accounting-leak-task-106-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-07-23T13:21:39Z
created_by: a-0b77j7k11m
about: [[084]]
---
# Filed the idle-cycle token-accounting leak (task 106) as the single highest-value change
## Chose
Filed the idle-cycle token-accounting leak (task 106) as the single highest-value change
## Rejected
the loop review spawn also ignoring --max-tokens on normal cycles; the loop 'wait' phase not scoping to the current cycle's wave and ignoring its timeout error; re-filing already-tasked findings (github-pull closed issues=104, loop force-close-on-refusal=102, loop seq-vs-priority ordering=103, git subprocess deadlines=105)
## Because
The already-tasked findings were excluded to avoid duplicate work. Among genuinely un-tasked defects, the idle-path token leak is highest value: it silently defeats --window-tokens — the loop's primary cost-control affordance — in exactly the Idle steady state where a long-running perpetual loop spends most of its wall-clock and tokens (empty backlog -> idle review regenerates work -> repeat). It is statically verifiable (windowSpent only grows in AfterCycle/governor.go:144, fed only by runCycle; the Idle branch at orchestration.go:304-313 bypasses both), and it is a real correctness gap that task 091 left open (091 wired accounting into runCycle ONLY). The normal-cycle review --max-tokens gap and the wait-timeout gap are real but lower-impact (bounded, or overlapping task 102), so they are folded in as secondary acceptance criteria / notes rather than competing tasks.
