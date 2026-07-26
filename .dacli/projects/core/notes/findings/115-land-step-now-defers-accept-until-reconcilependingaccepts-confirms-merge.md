---
id: f-115-land-step-now-defers-accept-until-reconcilependingaccepts-confirms-merge
kind: note
note_kind: finding
created: 2026-07-26T22:41:03Z
created_by: a-h7x65x1cg6
about: [[115]]
severity: moderate
---
# 115: LAND step now defers accept until reconcilePendingAccepts confirms merge (orchestration.go)
runCycle's --pr LAND step (orchestration.go, previously line ~476) no longer calls 'accept <ref> --force' immediately after a task's spawn succeeds. Instead it appends {Seq,Branch} to driver.pendingAccept. A new d.reconcilePendingAccepts(), called at the top of every loop() iteration right after d.syncTrunk(), classifies each pending task's branch via a new d.prLandStatus(branch) (gh pr list --json state first, falling back to a fresh 'git fetch origin <trunk>' + gitx.IsAncestor when gh finds no PR -- duplicated from vcs.checkLanded, not imported, per the feature-slice isolation rule): merged closes the task now (accept --force); orphaned (PR closed unmerged) drops it from pendingAccept so the still-open task re-enters the ready pool for a fresh attempt; landing/unknown keeps it pending. A new excludePending(tasks, pending) filters the ready frontier so a task with an in-flight PR is never rebuilt by the next cycle while its first PR is still live. Regression + unit tests added in driver_test.go: TestRunCycleLeavesRefusedSpawnTaskOpenButParksBuiltTaskPending, TestReconcilePendingAcceptsClosesOnConfirmedMerge, TestReconcilePendingAcceptsReopensOnClosedUnmergedPR, TestReconcilePendingAcceptsKeepsWaitingWhilePROpen, TestExcludePendingKeepsInFlightTaskOutOfReadyFrontier, TestLoopFullArcDefersAcceptThenClosesOnlyAfterMergeConfirmed. go build ./... clean; go test ./internal/... all green (incl. internal/cli's TestFeatureSlicesAreIsolated, confirming no vcs import).
