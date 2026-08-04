---
id: d-115-defer-accept-force-until-pr-merge-confirmed-tracked-via-in-memory
kind: note
note_kind: decision
created: 2026-07-26T22:40:51Z
created_by: a-h7x65x1cg6
about: [[115]]
github:
  issue: 352
  repo: mlnomadpy/dacli
---
# 115: defer accept --force until PR-merge confirmed, tracked via in-memory pendingAccept + excludePending
## Chose
115: defer accept --force until PR-merge confirmed, tracked via in-memory pendingAccept + excludePending
## Rejected
Marking the task active/in-review via a new task-status transition (dacli task claim) instead of leaving it open+tracked in-memory
## Because
readyTasks() only ever selects StatusOpen tasks, so simply never calling accept keeps the task open by construction; the only extra piece needed was excluding it from the ready frontier while a PR is in flight, which a driver-local pendingAccept list (mirroring the existing pendingLand precedent for the push-gate) does without any new store/status semantics or cross-slice command calls. reconcilePendingAccepts (checked every loop() iteration, right after syncTrunk) classifies each pending task's branch via gh PR state first, a fresh-fetch ancestor check as fallback -- duplicated from features/vcs.checkLanded's exact logic since arch_test's TestFeatureSlicesAreIsolated forbids orchestration importing vcs (same reasoning as the existing taskBranch/criticalPathSlack duplications).
