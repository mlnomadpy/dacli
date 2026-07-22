---
id: t-01KY53QHGE1D1EJ09FXG8733K4
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# E3: auto-claim on spawn so D1 calibration populates from real runs
## Context
E3 makes D1 calibration LIVE on real data. Today calibrate's by-agent-band is empty because spawned tasks have no `claimed by` Log stamp, so no claim→completion span exists to join against run records. Fix: stamp the claim at spawn.

Anchors:
- `internal/features/execution/execution.go` `cmdSpawn` — after the task `t` is resolved and the child id minted, append a claim stamp to the TASK if it isn't already claimed: `store.AppendLog(t, "claimed by "+childID)` (store.go:488, already exists) then `store.SaveTask(t)` (store.go:471). `internal/store/calibration.go:141` reads the FIRST "claimed by" stamp as the span start, so only stamp when none exists yet (idempotent — a re-spawn/supervise must not add a second claim). The task is loaded from the shared root, so the stamp lands there and travels with the task.
- Do this for both `cmdSpawn` and `cmdSupervise` (supervise already owns one child across turns — claim once on turn 1).
- This ONLY touches execution.go and calls existing store functions — do NOT edit store.go (E1 owns it this wave).

## Scope (STRICT) — touch ONLY:
- `internal/features/execution/execution.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY execution.go plus this task's file. `go build ./...` + `go test ./internal/...` green (execution has cli-driven tests; a spurious extra Log line could shift assertions — check). `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only; file a completion finding.

## Acceptance
- [x] spawn stamps a claim on the task at launch so a claim->done span exists; calibrate by-agent-band then joins run records to actuals and stops being empty on real agent data
- [x] no double-claim on re-spawn/supervise; existing claim is respected
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T14:52:58Z accepted by a-root
