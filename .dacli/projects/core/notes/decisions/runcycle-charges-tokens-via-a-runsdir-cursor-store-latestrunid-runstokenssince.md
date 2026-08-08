---
id: d-runcycle-charges-tokens-via-a-runsdir-cursor-store-latestrunid-runstokenssince
kind: note
note_kind: decision
created: 2026-07-23T10:23:39Z
created_by: a-s56ztpjbv1
about: [[091]]
---
# runCycle charges tokens via a RunsDir cursor (store.LatestRunID/RunsTokensSince), not a runner-returned value
## Chose
runCycle charges tokens via a RunsDir cursor (store.LatestRunID/RunsTokensSince), not a runner-returned value
## Rejected
having spawn/wait's runner interface return token counts directly, threaded up through runCycle
## Because
runner is an interface the driver calls per dacli-subcommand invocation (spawn, wait, ship, retro) and is faked in tests; dacli spawn --detach already returns immediately with no token total (wait finalizes and writes usage.txt asynchronously via writeUsage in execution.go), so there is no single call whose return value carries the cycle's tokens. Snapshotting store.LatestRunID(w) before the phase and summing store.RunsTokensSince(w, since) after reads the same usage.txt actuals calibration already trusts (store.readUsage), needs no runner/interface change, and naturally covers every spawn the cycle makes (build wave + review) since RunsDir entries are ULID-ordered
