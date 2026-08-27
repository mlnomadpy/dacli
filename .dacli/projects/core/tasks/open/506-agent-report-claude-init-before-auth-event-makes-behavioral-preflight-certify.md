---
id: t-01M1068KPR0JKWKP01WQNZEZD4
kind: task
created: 2026-08-26T23:25:10Z
created_by: a-root
owner: a-root
github:
  issue: 803
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] Claude init-before-auth event makes behavioral preflight certify an unauthenticated runtime
## Context
Adopted from GitHub issue #803.

Regression of #715 observed in bounded loop cycle 121. Runtime doctor for cc-rw and cc reported result=probed/compatible with 'exact adapter startup readiness event observed', and preflight returned no mismatches. The immediately following governed implementation run 01M1062SXY and review run 01M10630JD both exited in under one second with 'Not logged in · Please run /login' and produced zero child events. The installed Claude Code 2.1.198 emits the stream-json system/init event before its later authentication failure, while TestUnauthenticatedClaudeIsRefusedBeforeSpawn models only an immediate stderr failure. Manual workaround: stop the loop, avoid retrying unchanged, and continue through the owner/recovery worker. Expected: the Claude adapter handshake must not return compatible on init alone when the same startup then deterministically reports unauthenticated; doctor, preflight, spawn, and loop must agree and refuse before run/worktree creation. Add a fixture that emits init then Not logged in and mutation-prove the refusal.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] A deterministic Claude fixture emits `system/init` and then `Not logged in`, reproducing the false-compatible result from runs 01M1062SXY and 01M10630JD.
- [ ] Runtime doctor classifies that sequence as incompatible at the authentication layer and names `/login` as the recovery action.
- [ ] Preflight, spawn, and a bounded loop refuse before creating a governed run, task claim, or worktree for the unauthenticated runtime.
- [ ] An authenticated Claude startup remains bounded and compatible without waiting for a complete model turn.
- [ ] Mutation evidence proves that accepting `system/init` immediately makes the regression fail, and the full repository verification gates pass.
## Log
