---
id: f-task-369-regressed-dead-leader-descendant-monitoring-while-fixing-reused-groups
kind: note
note_kind: finding
created: 2026-08-12T14:07:27Z
created_by: a-codex-loop-auditor-yq4y7k
about: "[[303]]"
severity: major
---
# Task 369 regressed dead-leader descendant monitoring while fixing reused groups
Commit 6b417aa replaced runStillLive's AliveRecord-or-GroupAlive logic with procmon.ReconcileRun, which is exactly AliveRecord (internal/procmon/procmon.go:187), and deleted TestRunStillLiveDetectsLiveGroupWithDeadLeader. A legitimate leader that exits after forking a still-running commit/helper child is therefore reported dead immediately, reviving dacli-177's land-while-child-mid-commit failure. The prior invariant is explicit in done task 177 and task 285 acceptance; task 369 only claims descendant monitoring while the leader remains alive, which does not preserve the dead-leader case. Focused current RunStillLive/KillOne tests pass because none models a genuine surviving child.
