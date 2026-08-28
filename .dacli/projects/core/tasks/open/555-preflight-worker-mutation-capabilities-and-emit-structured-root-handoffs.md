---
id: t-01M1493JKBNQCJRES2HE15X3BF
kind: task
created: 2026-08-28T13:31:49Z
created_by: a-root
owner: a-root
github:
  issue: 874
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[t-01M147RA4B2C2NAH855VBNT2SJ, t-01M146BA07Z5BTS3TTB2ADW7D4]"
---
# Preflight worker mutation capabilities and emit structured root handoffs
## Context
Adopted from GitHub issue #874.

## Parent

Extracted from #871. Depends on the whole-cycle preflight contract in #867 and complements resumable recovery in #859.

## Observed symptom

A worker can edit and test source successfully while lacking permission to write worktree Git metadata or dacli events. The run may then be summarized as producing nothing even though valuable uncommitted work and verification evidence exist.

## Objective

Probe the concrete mutation capabilities required by the resolved assignment before launch and emit a structured root handoff whenever the worker cannot complete an authorized lifecycle step.



## Non-goals

- Automatically escalating host permissions.
- Treating worker prose as structured evidence.
- Giving read-only reviewers commit authority.

## Manual workaround today

The owner inspects the worker worktree, reruns tests, reclaims it, commits, and reconstructs the missing task/event history manually.

## Acceptance
- [ ] Preflight probes, without corrupting state, source-path writes within claims, worktree Git metadata/lock writes, dacli event writes, required network access, and declared runtime/package-manager executability.
- [ ] The result distinguishes policy refusal, filesystem sandbox refusal, authentication/network failure, missing tool, and transient contention.
- [ ] A required permanent capability failure refuses before worker creation; a capability that is intentionally delegated to root is represented as a planned handoff rather than worker success.
- [ ] When edited or verified work exists but commit/event publication is unavailable, the run emits a versioned handoff containing task/run IDs, exact changed paths, diff/tree digest, verification commands/results, unresolved findings, failed operation and stderr, and safe owner next action.
- [ ] Root can consume the handoff only after re-observing the worktree and hashes; stale or changed material refuses.
- [ ] `wait`, `agents`, reconciliation, and loop recovery report `handoff-required`, never `produced nothing`, for this state.
- [ ] Harness pinning and least-authority grants remain intact; recovery does not silently switch provider or grant broader worker authority.
- [ ] Fixtures cover source-write refusal, Git-index refusal after successful tests, event-write refusal, and stale handoff consumption.
## Log
- 2026-08-28T13:32:49Z dependency edit by a-root (event 01M1495D0N24B5CBF6AE87WRR0)
