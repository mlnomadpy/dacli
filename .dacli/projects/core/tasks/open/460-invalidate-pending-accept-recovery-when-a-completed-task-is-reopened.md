---
id: t-01M05Y78XTYNQ4GP3AV396JFD6
kind: task
created: 2026-08-16T18:44:23Z
created_by: a-root
owner: a-root
priority: must
github:
  issue: 679
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Invalidate pending-accept recovery when a completed task is reopened
## Context
Adopted from GitHub issue #679.

## Symptom

After task 452 / issue #657 was accepted and its PR merged, a live post-merge proof found an uncovered defect. The owner correctly ran:

```
dacli task reopen 452 --reason "..."
```

The task moved back to open and its acceptance boxes were cleared. The next bounded loop still restored the old merged `pending_accept` journal entry:

```
452: PR merged but command acceptance requires verifier evidence — run `dacli accept 452 --verify "<command>"`; keeping the recovery entry
```

Because `excludePending` treats that entry as in-flight, the loop skipped the reopened must-priority task and selected task 441 instead. It also reclaimed task 452's worktree as merged, even though the task had deliberately been reopened for corrective work.

## Expected

Reopening a task invalidates any prior landing/accept recovery entry for that task. The reopened task must return to the ready frontier and its worktree must not be reclaimed as completed work solely because the earlier PR merged.

## Proven cause

The cycle journal is independent from canonical task reopening. The state-aware reconciliation added for #661 recognizes done/accepted tasks and open command-verification tasks, but has no durable signal distinguishing a deliberately reopened task from an open task merely awaiting owner verification after merge.

## Manual workaround

Remove only task 452's stale `pending_accept` journal entry after the active loop finishes, then rerun the loop. No safe public dacli command currently performs this targeted reconciliation.

## Design

Make reopening invalidate merge-recovery state structurally. Prefer a task generation/reopen marker that reconciliation can compare with the journal entry; do not infer intent merely from open status. A reopened task may legitimately reuse the same sequence and branch but represents a new corrective work generation.

## Acceptance

- [ ] A regression starts with a done, accepted task and persisted merged `pending_accept`, then reopens the task through the public task command.
- [ ] The next loop reconciliation removes or invalidates the prior-generation pending entry and does not emit the verifier-required recovery message.
- [ ] The reopened task is present in the ready frontier and can be selected according to its priority.
- [ ] Worktree pruning does not reclaim a reopened task checkout solely because its prior PR merged.
- [ ] An open task that has not been reopened and genuinely awaits verifier evidence retains the #661 behavior.
- [ ] Journal format migration is backward compatible or explicitly documented.
- [ ] Mutation evidence, focused orchestration/planning tests, and `go test ./...` pass.

## Acceptance
## Log
- 2026-08-17T15:31:04Z claimed by a-maintainer-z795wm
