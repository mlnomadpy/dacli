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
- [ ] an agent spawned with --worktree writes its code into that worktree and cannot modify the main checkout
- [ ] the mechanism is named: whether it is cwd, the brief's paths, or the preamble that sends the agent to the wrong root
- [ ] a test spawns into a worktree and asserts the main checkout is unchanged afterwards
## Log
