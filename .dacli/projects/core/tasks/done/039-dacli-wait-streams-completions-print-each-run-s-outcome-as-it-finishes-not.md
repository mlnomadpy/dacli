---
id: t-01KY574DHF5V0S388C3CVK62A0
kind: task
created: 2026-07-22T15:29:28Z
created_by: a-root
owner: a-root
priority: could
estimate: {optimistic: 1, probable: 2, pessimistic: 3}
---
# dacli wait streams completions: print each run's outcome as it finishes, not silent until return
## Context
NOTE: `cmdWait` ALREADY streams — `internal/features/execution/execution.go` `cmdWait` (~:1318) prints `finalizeRun` for each run the moment its process is detected gone (the `for len(pending) > 0` loop, ~:1356). The "silent" complaint was mostly an artifact of the wait being backgrounded by the caller. So this is a small UX polish, not a rewrite:
- Print a startup line naming how many runs it is waiting on (e.g. `waiting on N run(s): <ids>`).
- Between completions the loop is silent for the whole `interval` gap; add a light heartbeat so a long wait shows life — e.g. every ~30s print `still waiting on K run(s) (up <elapsed>)`. Do NOT spam every poll.
- Keep the existing per-completion streaming exactly as is.

Keep it minimal and quiet; the goal is "a foreground wait shows progress", not verbose logging.

## Scope (STRICT) — touch ONLY:
- `internal/features/execution/execution.go` (only cmdWait)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY execution.go plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [x] dacli wait prints each run's completion line the moment it finishes (child done, N of M) rather than blocking silently until all are done
- [x] committed on branch by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T15:30:44Z claimed by a-87qa0eamp1
- 2026-07-22T15:38:12Z accepted by a-root
- 2026-07-22T15:38:12Z completed by a-root
- 2026-07-22T16:17:27Z status done proposed by a-87qa0eamp1, applied (event 01KY578NN3A74H3BD9XAAKC789)
