---
id: f-supervise-recovery-still-launches-from-shared-root-when-owner-invokes-it-there-2f9ndv
kind: note
note_kind: finding
created: 2026-08-28T10:36:50Z
created_by: a-adversarial-reviewer-12hhsd
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# Supervise recovery still launches from shared root when owner invokes it there
internal/features/execution/execution.go:1647-1651 asks resolveSpawnWorkDir to preserve a recovered task checkout, but execution.go:3626-3635 returns w.Root immediately whenever cmdSupervise is invoked from the shared checkout, before inspecting registered worktrees or worktree-transfer evidence. Trigger: root invokes supervise from the main checkout after reclaiming a failed child's task worktree. Wrong outcome: correction turns run in main with isolatedWorktree=false and no task worktree run record, so governed commits lack the task-scoped ownership task 532 was meant to restore. internal/features/execution/spawn_worktree_test.go:121-156 invokes cmdSupervise with newCtx(wt), so it cannot catch the root-invoked path. Open and active core task lists contain no task for this mismatch; completed task 532 claimed this scenario but its shipped regression does not reproduce it. Focused verification: GOCACHE=/tmp/dacli-audit-cache go test ./internal/features/execution -run 'TestSuperviseCorrectionResumesRootReclaimedTaskWorktreeAcrossTurns|TestResolveSpawnWorkDir' -count=1 passed, confirming current tests do not expose the gap.
