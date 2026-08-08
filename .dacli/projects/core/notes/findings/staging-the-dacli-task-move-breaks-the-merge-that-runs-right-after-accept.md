---
id: f-staging-the-dacli-task-move-breaks-the-merge-that-runs-right-after-accept
kind: note
note_kind: finding
created: 2026-08-04T20:20:45Z
created_by: a-maintainer-yd9g1p
about: "[[273]]"
severity: major
---
# Staging the .dacli task-move breaks the merge that runs right after accept
gitx.Merge (lifecycle.go:1012) runs 'git merge --no-ff' which tolerates an UNSTAGED dirty .dacli (IsCleanExcept) but REFUSES once the move is staged: 'Your local changes to .dacli/tasks/.../*.md would be overwritten by merge' (verified in a temp repo AND via internal/gitx test). This holds even when staging ONLY the deletion of the old path. So any 'accept stages its own record move' fix at MoveTask/index level regresses dacli ship/merge (ship order: accept -> merge -> commitRecord). The move must instead be COMMITTED (clean tree) or reconciled by doctor, not left staged.
