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
- [x] A regression starts with a done, accepted task and persisted merged `pending_accept`, then reopens the task through the public task command.
- [x] The next loop reconciliation removes or invalidates the prior-generation pending entry and does not emit the verifier-required recovery message.
- [x] The reopened task is present in the ready frontier and can be selected according to its priority.
- [x] Worktree pruning does not reclaim a reopened task checkout solely because its prior PR merged.
- [x] An open task that has not been reopened and genuinely awaits verifier evidence retains the #661 behavior.
- [x] Journal format migration is backward compatible or explicitly documented.
- [x] Mutation evidence, focused orchestration/planning tests, and `go test ./...` pass.
## Acceptance
## Log
- 2026-08-17T15:31:04Z claimed by a-maintainer-z795wm
- 2026-08-17T16:06:40Z accepted by a-root
- 2026-08-17T16:06:40Z verified by `GOCACHE=/tmp/dacli-460-owner-final go test ./...` (exit 0) in branch main at 0225ea7 — proves that tree builds, not that the work is in trunk
- 2026-08-17T16:06:40Z deliverable: dacli/460-invalidate-pending-accept-recovery-when-a-completed-task-is-reopened is merged into main
- 2026-08-17T16:06:40Z completed by a-root
- 2026-08-17T16:13:37Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/681 (event 01M086KWBB4Y6EH7SEY46AP8FN)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-460-owner-check go test ./...","exit_code":0,"duration_ms":72212,"artifact_hash":"sha256:491bf8912b4ad18ff1d6da1e9c204c9f4a3679038c2ed998f8e1291ba21d862d","verifier":"a-root","branch":"main","commit_sha":"0225ea7cb39f67541395dbe1b544b015d9ef0aee"}
{"command":"GOCACHE=/tmp/dacli-460-owner-final go test ./...","exit_code":0,"duration_ms":81743,"artifact_hash":"sha256:f0a5d5b03d0b24935f7a3bd165138790ab62ecab09079be4b2d1a9904da07fd9","verifier":"a-root","branch":"main","commit_sha":"0225ea7cb39f67541395dbe1b544b015d9ef0aee"}
