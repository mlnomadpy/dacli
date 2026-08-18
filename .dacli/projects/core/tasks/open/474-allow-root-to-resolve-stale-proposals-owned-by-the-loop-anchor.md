---
id: t-01M0AKHSFGWWSMDFCWCE9RYCGQ
kind: task
created: 2026-08-18T14:14:05Z
created_by: a-root
owner: a-root
github:
  issue: 699
  repo: mlnomadpy/dacli
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Allow root to resolve stale proposals owned by the loop anchor
## Context
Adopted from GitHub issue #699.

## Symptom

`dacli events pending` lists eight historical `propose-status` events targeting the standing continuous-improvement task. Their actors are retired loop-auditor agents, but the task owner is the synthetic identity `loop`. `dacli sync` applies none of them. `dacli events dismiss <id> --reason ...` as `a-root` refuses with exit 3 and `refused-unrelated`. The events therefore remain permanently pending.

## Reproduction

1. Create or use a standing loop task owned by the synthetic `loop` identity.
2. Have a loop reviewer propose a terminal task status, then retire/finalize the reviewer without applying it.
3. Run `dacli sync` as root; observe zero applied.
4. Run `dacli events dismiss <event> --reason stale` as root; observe exit 3 with `refused-unrelated`.
5. Run `dacli events pending`; observe the impossible proposal remains pending.

## Suspected cause

`dismissalAuthority` permits root orphan recovery only when `agentRetired(w, task.Owner())` is true. The loop anchor owner is the synthetic value `loop`, which is not an agent record and can never become retired. The readout says `refused-unrelated` even though root owns the workspace and no identity can resolve the proposal.

## Risk

Pending-event counts permanently disagree with actionable work, mature workspaces accumulate impossible proposals, and operators cannot return the event queue to a truthful state without manually altering append-only records.

## Manual workaround

None through the supported CLI. Preserve the pending events and ignore the inflated count. Do not edit or delete append-only event files manually.

## Design direction

Model synthetic task ownership explicitly. Let rw root reject an unapplied proposal targeting a recognized synthetic loop-owned task when the actor is terminal/retired and no live run can resolve it. Preserve fail-closed behavior for ordinary live owners and unresolved/corrupt targets. Record an audited dismissal rather than modifying history.

## Acceptance
- [ ] A public-command regression creates a proposal targeting a task owned by `loop` and proves rw root can dismiss it with a non-empty reason.
- [ ] The original event remains append-only history and one dismissal event records root, target event ID, and reason.
- [ ] The dismissed event disappears from `events pending` and no longer inflates overview pending counts.
- [ ] Root is still refused for an unrelated proposal whose ordinary task owner is live.
- [ ] Read-only identities cannot dismiss another actor’s proposal.
- [ ] Unresolved/corrupt targets remain fail-closed.
- [ ] Mutation evidence, focused collab/eventlog tests, race tests, and `go test ./...` pass.
## Log
- 2026-08-18T14:29:38Z claimed by a-fixer-rmqgbs
- 2026-08-18T15:14:58Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/704 (event 01M0AMY4ABWXZR5V65RZ0DBEHP)
