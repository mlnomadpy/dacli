---
id: f-102-complete-on-branch-dacli-102-runcycle-tracks-per-task-spawn-outcome-pr-land
kind: note
note_kind: finding
created: 2026-07-23T13:28:57Z
created_by: a-c7sr25jttk
about: [[102]]
severity: moderate
---
# 102 complete on branch dacli/102-...: runCycle tracks per-task spawn outcome, --pr LAND only force-closes tasks whose spawn actually built
Commit f8c22e4 by a-c7sr25jttk (fixer), staged only internal/features/orchestration/{orchestration.go,driver_test.go}. runCycle (orchestration.go) now tracks a built[seq]=bool per batch task: false if the spawn command itself errors (synchronous refusal), and re-checked false after wait if the task's dacli/<seq>-slug branch does not exist (async failure — child crashed/killed without committing). taskBranch() duplicates vcs.BranchFor's dacli/%03d-%s convention locally since orchestration must not import vcs (TestFeatureSlicesAreIsolated). branchExists() checks both refs/heads and refs/remotes/origin. The --pr LAND step's accept --force loop (orchestration.go ~L422-429) now skips any task with built[seq]==false, logging 'spawn refused/failed — leaving open for retry' instead of closing it — so the next cycle's readyTasks() re-picks it. New regression TestRunCycleLeavesRefusedSpawnTaskOpenButClosesSucceeded (driver_test.go) drives a 2-task batch through a spawnOutcomeRunner where one task's spawn returns an error and never gets a branch; asserts accept --force is never called for that task, it remains in the open backlog, while its successfully-spawned sibling (real branch created) gets accept --force called and moves to done. go build ./... clean; go test ./internal/... all green (incl. orchestration, cli, store). Box-checking refused for non-owner (only a-534c4gav5p) — owner should verify and close via dacli task check/done + dacli merge --task 102.
