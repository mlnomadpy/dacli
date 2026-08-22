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
depends_on: "[t-01M0MZ05DNTWBYQSTYP5X8J2NF]"
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
- [x] A fixture with zero live runs and historical non-retired identities makes `dacli agents` and `dacli team` agree that live occupancy is zero.
- [x] A fixture with one live run increments exactly that role’s WIP occupancy and reduces headroom once.
- [x] Finished, failed, killed, and never-started historical identities do not consume live WIP capacity.
- [x] Role removal retains the fail-closed provenance behavior delivered by #690; its predicate is not weakened to match WIP occupancy.
- [x] CLI help/docs name the distinction between identity history, current liveness, and WIP occupancy.
- [x] Dashboard/API roster projections use the same shared occupancy predicate as `dacli team`.
- [x] A regression test demonstrates the previous contradiction by breaking the new predicate and observing the test fail.
## Log
- 2026-08-22T14:46:45Z dependency edit by a-root (event 01M0MZ0FKPDR77W2EA08BW3ZZJ)
- 2026-08-22T15:19:57Z claimed by a-maintainer-qt88ce
- 2026-08-22T15:49:50Z accepted by a-root
- 2026-08-22T15:49:50Z verified by `GOFLAGS=-mod=readonly GOCACHE=/tmp/dacli-go-cache-473-trunk go test ./internal/store ./internal/features/teamops ./internal/features/dashboard ./internal/features/execution ./internal/cli -run 'TestNeverStartedAgentDoesNotConsumeLiveOccupancy|TestLiveOccupancyUsesRunRoleDespiteRetirement|TestAgentSpawnWIPLimit|TestTeamRosterReportsHeadroomAndUnroledAgents|TestRoleWiredIntoSpawn|TestAPIRoles|TestAPIStateEmbedsRoles|TestRolesEmptyWorkspaceIsZeroSafe' -count=1` (exit 0) in branch main at a931b1a — proves that tree builds, not that the work is in trunk
- 2026-08-22T15:49:50Z deliverable: dacli/473-separate-historical-agent-identities-from-live-roster-occupancy is merged into main
- 2026-08-22T15:49:50Z completed by a-root
- 2026-08-22T16:37:47Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/770 (event 01M0N243VKRNCT94K8NWW20AMV)
- 2026-08-22T16:37:47Z a-root: Landing policy override: mode=pr base=main (event 01M0N2FQPE8WG84ZZTJBCADEAS)
- 2026-08-22T16:37:47Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/770 at merge commit a931b1a81603d22799b98b9db64bad8f26a7454f into main (event 01M0N2HTP0B6M58CJ2NCR5S1SK)
## Verification Evidence
{"command":"GOFLAGS=-mod=readonly GOCACHE=/tmp/dacli-go-cache-473-trunk go test ./internal/store ./internal/features/teamops ./internal/features/dashboard ./internal/features/execution ./internal/cli -run 'TestNeverStartedAgentDoesNotConsumeLiveOccupancy|TestLiveOccupancyUsesRunRoleDespiteRetirement|TestAgentSpawnWIPLimit|TestTeamRosterReportsHeadroomAndUnroledAgents|TestRoleWiredIntoSpawn|TestAPIRoles|TestAPIStateEmbedsRoles|TestRolesEmptyWorkspaceIsZeroSafe' -count=1","exit_code":0,"duration_ms":5168,"artifact_hash":"sha256:0f6ed02ddbe1b571c1c096291398a3e9834b98a45a2f2294b0720b37b3b26793","verifier":"a-root","branch":"main","commit_sha":"a931b1a81603d22799b98b9db64bad8f26a7454f"}
{"command":"GOFLAGS=-mod=readonly GOCACHE=/tmp/dacli-go-cache-473-trunk go test ./internal/store ./internal/features/teamops ./internal/features/dashboard ./internal/features/execution ./internal/cli -run 'TestNeverStartedAgentDoesNotConsumeLiveOccupancy|TestLiveOccupancyUsesRunRoleDespiteRetirement|TestAgentSpawnWIPLimit|TestTeamRosterReportsHeadroomAndUnroledAgents|TestRoleWiredIntoSpawn|TestAPIRoles|TestAPIStateEmbedsRoles|TestRolesEmptyWorkspaceIsZeroSafe' -count=1","exit_code":0,"duration_ms":5936,"artifact_hash":"sha256:57c6f716dc6e002fba4b7b323e64923bfeda81bd03f5081bc43339a2d9fc3a5b","verifier":"a-root","branch":"main","commit_sha":"a931b1a81603d22799b98b9db64bad8f26a7454f"}
