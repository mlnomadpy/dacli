---
id: t-01KY4YCXP62TQ9VSMADBCH18V2
kind: task
created: 2026-07-22T12:56:50Z
created_by: a-root
owner: a-root
priority: could
estimate: {optimistic: 1, probable: 2, pessimistic: 3}
---
# Gate replay metadata reads, hoist FindTask, surface writeRun errors
## Context
Real audit finding. Anchors:
- `internal/features/execution/replay.go:82-91` — the loop over run dirs calls `readRunMeta` (opens invocation.txt + reads outcome.md, 2 file opens) for EVERY run, then in the id-prefix branch (`case len(f.Pos) > 0 && strings.HasPrefix(...)`) discards it for non-matching names. Gate the `readRunMeta` call behind the cheap `strings.HasPrefix(e.Name(), f.Pos[0])` so a single-run replay doesn't read metadata for the whole runs dir.
- `replay.go:91` calls `store.FindTask(w, taskRef)` inside the loop although `taskRef` is loop-invariant — resolve it ONCE before the loop and compare `m.taskID` to the cached `t.ID`.
- `internal/features/execution/execution.go:276` `writeRun` does `_ = os.WriteFile(...)` for the replay-capture brief/invocation/outcome — a failed write is swallowed and later surfaces as "brief not recorded" with no hint. Surface the error (e.g. warn to stderr) instead of dropping it. Do NOT touch the agent-file copy write at :323 (best-effort by design).

## Scope (STRICT) — touch ONLY:
- `internal/features/execution/replay.go`
- `internal/features/execution/execution.go` (only the writeRun helper)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the two files above plus this task's file. `go build ./...` + `go test ./internal/...` green (cli TestMain clears DACLI_AGENT). Paste summary as `dacli note add finding`, `dacli commit`, then `dacli task check`.

## Acceptance
- [x] replay reads run metadata only for matching run dirs (gate readRunMeta behind the id-prefix check), not every dir in single-prefix mode
- [x] replay resolves the loop-invariant taskRef via FindTask ONCE before the loop
- [x] writeRun surfaces a failed run-record write instead of silently swallowing it
- [x] committed on branch by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T13:03:24Z completed by a-root
