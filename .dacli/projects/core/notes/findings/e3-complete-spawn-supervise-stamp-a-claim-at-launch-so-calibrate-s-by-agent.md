---
id: f-e3-complete-spawn-supervise-stamp-a-claim-at-launch-so-calibrate-s-by-agent
kind: note
note_kind: finding
created: 2026-07-22T14:43:23Z
created_by: a-yennmqf72n
about: [[033]]
severity: moderate
---
# E3 complete: spawn/supervise stamp a claim at launch so calibrate's by-agent band populates
execution.go: new claimTask(ctx,t,childID) helper appends 'claimed by <childID>' to the task Log via store.AppendLog + store.SaveTask, but ONLY when no 'claimed by' exists in the Log section (idempotent — respects the first owner on re-spawn/supervise). Wired into cmdSpawn right after agentid.Spawn mints childID, and into cmdSupervise once before the turn loop (one child owns the task across turns). This is the span start calibration.logSpan (calibration.go:141) reads FIRST, so a claim->completed span now exists for real spawned runs and calibrate by-agent-band stops being empty. Scope: execution.go only; store.go untouched (E1 owns it). go build ./... clean; go test ./internal/... all green (cli suite included — no assertion shift from the extra Log line).
