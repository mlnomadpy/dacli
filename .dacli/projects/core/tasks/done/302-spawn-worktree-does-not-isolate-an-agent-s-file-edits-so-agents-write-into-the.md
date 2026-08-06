---
id: t-01KZ7EQGGWV8JS8J346KD2XCFQ
kind: task
created: 2026-08-04T22:36:24Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# spawn --worktree does not isolate an agent's file edits, so agents write into the main checkout
## So that
parallel agents cannot break trunk for everyone, which is the entire premise of running a wave
## Acceptance
- [x] an agent spawned with --worktree writes its code into that worktree and cannot modify the main checkout
- [x] the mechanism is named: whether it is cwd, the brief's paths, or the preamble that sends the agent to the wrong root
- [x] a test spawns into a worktree and asserts the main checkout is unchanged afterwards
## Log
- 2026-08-05T13:08:50Z claimed by a-fixer-zrcq2v
- 2026-08-05T13:34:18Z accepted by a-root
- 2026-08-05T13:34:18Z verified by `go test ./internal/gitx/... ./internal/cli/... ./internal/features/execution/...` (exit 0)
- 2026-08-05T13:34:18Z completed by a-root
