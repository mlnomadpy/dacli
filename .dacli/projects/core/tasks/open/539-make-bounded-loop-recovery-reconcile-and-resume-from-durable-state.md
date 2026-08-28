---
id: t-01M146BA07Z5BTS3TTB2ADW7D4
kind: task
created: 2026-08-28T12:43:36Z
created_by: a-root
owner: a-root
github:
  issue: 859
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
depends_on: "[t-01M146BA62817V08T9P6D6REKT, t-01M146BA26TEH86YE16XHDZKGY, t-01M11HZW5KX6JN902621F6F0ZH, t-01M11HZW8X678270CFWDBK7YTN, t-01M11HZWC4XH41GVP71EJS8Z4V, t-01M12N9W3XJC7JRKHCFGRT0XNG, t-01M12QX9HEPKAAS1033W6HS45D]"
---
# Make bounded-loop recovery reconcile and resume from durable state
## Context
Adopted from GitHub issue #859.

## Parent and prerequisites

Part of #855. Depends on the canonical reconciliation projection and PR/CI classifier. Coordinate with #813, #814, #811, #838, and #841 rather than duplicating their fixes.

## Observed symptom

The loop journal can retain stale pending landings or trunk observations while live agents, tasks, PRs, and GitHub disagree. `loop status` describes the persisted checkpoint but does not explain the authoritative blocker or prove that rerunning is safe.

## Objective

Make every bounded loop halt self-explanatory and resumable from durable, freshly reconciled state.

## Required behavior

- Reconcile the previous cycle before selecting or landing new work.
- Emit a typed halt class such as policy refusal, external blocker, inconsistent record, no schedulable work, budget exhaustion, or transient infrastructure failure.
- List the exact affected task/agent/run/PR/check/event identifiers and the minimum human or automated remedy.
- Automatically recognize real external-condition changes or trunk advancement on the next bounded invocation.
- Preserve STOP, budgets, WIP, claims, and landing authority across restart.
- Never clear a no-progress counter or stale record merely to make the loop continue.



## Non-goals

- An unbounded daemon or permissionless release system.
- Retrying exit-code 3 refusals unchanged.
- Bypassing branch protection or required checks.

## Manual workaround today

Operators inspect every relevant surface, finalize runs/events manually, diagnose PR/CI externally, and rerun a bounded loop after deciding the persisted checkpoint is safe.

## Acceptance
- [ ] Crash/restart fixtures cover interruption after spawn, wait, verification, push, PR creation, pending CI, merge, and record acceptance without duplicate execution or false completion.
- [ ] Closed-unmerged PR, missing canonical PR, account-level CI restriction, stale loop landing, and late auto-merge each produce distinct structured diagnoses.
- [ ] After a fixture changes from external blocker to observable green/merged state, the next bounded invocation resumes from the correct checkpoint without manual counter reset.
- [ ] A blocked or claim-conflicting task is not counted as schedulable when no worker can safely claim it.
- [ ] Terminal-task events and finished runs are finalized or surfaced for explicit reconciliation before a new wave.
- [ ] JSON status includes cycle/checkpoint, halt class, affected refs, observed trunk/PR/check state, retryability, and next action.
- [ ] Mutation tests fail when the pre-cycle reconciliation gate is bypassed or unknown GitHub state is treated as success.
## Log
- 2026-08-28T12:44:47Z dependency edit by a-root (event 01M146DEXN8WR2EC11C3BHC0AH)
