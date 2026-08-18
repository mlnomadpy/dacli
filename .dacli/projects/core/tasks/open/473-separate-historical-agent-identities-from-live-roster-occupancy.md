---
id: t-01M0AK4XK4M7CTJ6DXRKFW8XWG
kind: task
created: 2026-08-18T14:07:03Z
created_by: a-root
owner: a-root
github:
  issue: 697
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Separate historical agent identities from live roster occupancy
## Context
Adopted from GitHub issue #697.

## Problem

On a mature workspace, `dacli agents` reports `no live agents` while `dacli team` reports dozens of agents as `active` (for example fixer:61, go-auditor:67, maintainer:66). The team view is counting minted, non-retired historical identities rather than processes/runs that occupy WIP now. This makes the operator-facing word `active` disagree with the live-agent command, obscures whether a role is actually saturated, and makes provider/model capacity decisions hard to trust.

This is distinct from #690: that issue correctly made role removal fail closed for identities that were minted but never ran. The read model still needs to distinguish conservative removal provenance from current execution occupancy.

## Reproduction

1. Use a mature workspace containing historical agent identities with completed or absent runs.
2. Run `dacli agents`; observe `no live agents`.
3. Run `dacli team`; observe non-zero `active` counts for multiple roles.
4. Compare WIP/headroom language with the process truth reported by `agents`.

## Suspected design cause

The roster read model appears to reuse identity retirement state as a proxy for liveness. Identity provenance, role-removal safety, current process liveness, and WIP occupancy are different concepts and need separate predicates.

## Design direction

Introduce one shared, named occupancy predicate derived from current run/process state and use it for team WIP/headroom. Preserve the stricter minted/live/finished predicate used by role-removal safety. Expose historical non-retired identities separately if they remain operationally useful, and make terminology consistent across `team`, `agents`, dashboard/API projections, and docs.



## Audit evidence

Observed in the dacli dogfood workspace on 2026-08-18: `dacli agents` printed `no live agents`; `dacli team` simultaneously printed fixer active:61, go-auditor active:67, maintainer active:66, plus other non-zero role counts.

## Acceptance
- [ ] A fixture with zero live runs and historical non-retired identities makes `dacli agents` and `dacli team` agree that live occupancy is zero.
- [ ] A fixture with one live run increments exactly that role’s WIP occupancy and reduces headroom once.
- [ ] Finished, failed, killed, and never-started historical identities do not consume live WIP capacity.
- [ ] Role removal retains the fail-closed provenance behavior delivered by #690; its predicate is not weakened to match WIP occupancy.
- [ ] CLI help/docs name the distinction between identity history, current liveness, and WIP occupancy.
- [ ] Dashboard/API roster projections use the same shared occupancy predicate as `dacli team`.
- [ ] A regression test demonstrates the previous contradiction by breaking the new predicate and observing the test fail.
## Log
