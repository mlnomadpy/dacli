---
id: f-supervise-recovery-still-launches-from-shared-root-when-owner-invokes-it-there
kind: note
note_kind: finding
created: 2026-08-28T09:59:01Z
created_by: a-adversarial-reviewer-4fgfsv
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# Supervise recovery still launches from shared root when owner invokes it there
internal/features/execution/execution.go:1647-1651 says a correction launched from root must preserve the registered task worktree, but cmdSupervise passes ctx.Cwd with explicit=false; resolveSpawnWorkDir at execution.go:3622-3629 immediately returns w.Root for any main-checkout caller without inspecting the task's registered worktree or durable worktree-transfer record. Trigger: the root owner invokes supervise from the shared checkout after reclaiming a failed child's task worktree. Wrong outcome: every correction runtime executes in main, records no worktree.txt, and its governed commit is refused for lacking the task worktree ownership that task 532 intended to restore. The regression at spawn_worktree_test.go:121-156 does not reproduce that trigger because it constructs newCtx(wt), already inside the linked worktree; the existing main-checkout behavior at execution.go:3622-3629 therefore remains untested for recovered supervision. Open/active core lists contain no task for this root-invoked supervise mismatch; task 531 is transcript creation and is distinct. Focused verification passed: GOCACHE=/tmp/dacli-audit-cache go test ./internal/features/execution ./internal/cli -count=1.
