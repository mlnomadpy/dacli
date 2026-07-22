---
id: t-01KY53QHH3SQP4BEWPK55BZD7N
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: could
estimate: {optimistic: 1, probable: 2, pessimistic: 3}
---
# E5: dacli agents --tail — last transcript line per live agent (thinking vs hung)
## Context
E5 closes the "is it thinking or hung" blind spot: `dacli agents` shows RAM/CPU but a reasoning agent and a wedged one look identical.

Anchors:
- `internal/features/execution/execution.go` `cmdAgents` — add a `--tail` flag. For each live agent, read the most recent non-empty line of its `w.RunDir(rec.RunID)/transcript.log` and print it (truncated, e.g. 100 chars) beneath or beside the resource line. A detached child streams straight to that file, so the last line is its current activity.
- Reuse the existing `lastLines` helper already in this file (added for `dacli logs`), or read the tail directly. Keep the default `dacli agents` output unchanged; `--tail` is additive.

## Scope (STRICT) — touch ONLY:
- `internal/features/execution/execution.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY execution.go plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` summary, then `dacli commit` (E2's claim-scoped check is now live — your commit must stay within this claim). Box-checking is owner-only.

## Acceptance
- [x] dacli agents --tail shows each live agent's most recent transcript line beside its RAM/CPU, so a working agent is distinguishable from a hung one without manually tailing files
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T14:53:55Z claimed by a-3x4vsezdhm
- 2026-07-22T14:57:13Z accepted by a-root
