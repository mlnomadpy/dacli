---
id: t-01KZXVTQ84PH0VT0NYEKM7Q2MX
kind: task
created: 2026-08-13T15:28:39Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 588
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Release path claims when detached runs finish or agents retire
## Context
Adopted from GitHub issue #588.

## Symptom

After run `01KZXVDSMM` finished and `dacli agents --tail` reported `no live agents`, `dacli agent retire a-fixer-j57wh6` succeeded but a new spawn for the same path still returned exit 3:

```
path-claim conflict: live agent a-fixer-j57wh6 already claims "internal/features/execution"
```

`dacli agent show a-fixer-j57wh6` simultaneously reported `retired: yes` and the prior run had a terminal `done` exit event.

## Impact

A completed worker permanently fences a path until an undocumented/manual cleanup, preventing evidence-informed follow-up turns. The supervisor can report no live agents while its claim gate reports the opposite.

## Reproduction

1. Spawn a detached worktree agent with `--claim internal/features/execution`.
2. Wait until the run is terminal and `agents` reports no live agents.
3. Retire the agent.
4. Spawn another worker with the same claim.
5. Observe exit 3 naming the retired worker as live.

## Acceptance criteria

- [ ] A terminal run releases its claims atomically with terminal outcome persistence.
- [ ] Retiring a terminal agent is idempotent and leaves no live claim.
- [ ] `agents`, spawn claim checks, and `agent show` use one liveness classification.
- [ ] A truly live worker still blocks an overlapping claim.
- [ ] Regression test reproduces terminal/retired agent conflict before the fix and permits the follow-up afterward.
- [ ] Crash recovery removes stale claims only after proving the recorded process identity is no longer live.

## Acceptance
- [x] A terminal run releases its claims atomically with terminal outcome persistence
- [x] Retiring a terminal agent is idempotent and leaves no live claim
- [x] agents, spawn claim checks, and agent show use one liveness classification
- [x] A truly live worker still blocks an overlapping claim
- [x] A regression test reproduces the terminal or retired agent conflict before the fix and permits the follow-up afterward
- [x] Crash recovery removes stale claims only after proving the recorded process identity is no longer live
## Log
- 2026-08-13T15:37:54Z claimed by a-fixer-dt88p4
- 2026-08-13T15:45:07Z accepted by a-root
- 2026-08-13T15:45:07Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T15:45:07Z deliverable: dacli/422-release-path-claims-when-detached-runs-finish-or-agents-retire exists but is NOT in main — closed anyway
- 2026-08-13T15:45:07Z completed by a-root
- 2026-08-13T16:16:39Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/590 (event 01KZXWS23Y24W09R8B7Q2NKNRC)
- 2026-08-13T16:16:39Z status done proposed by a-fixer-dt88p4, applied (event 01KZXWVY6RG4CDARRFH0GMM4EB)
