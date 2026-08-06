---
id: d-closetask-commits-its-scoped-record-move-not-stage-so-close-is-self-sufficient
kind: note
note_kind: decision
created: 2026-08-04T20:26:23Z
created_by: a-maintainer-yd9g1p
about: "[[273]]"
---
# CloseTask COMMITS its scoped record move (not stage), so close is self-sufficient without breaking the post-accept merge
## Chose
CloseTask COMMITS its scoped record move (not stage), so close is self-sufficient without breaking the post-accept merge
## Rejected
Staging the move in MoveTask/accept (my first attempt), or having doctor merely offer a manual fix
## Because
Proven: any STAGED .dacli move makes the 'git merge --no-ff' that dacli integrate/ship runs right after accept refuse ('local changes would be overwritten'), even staging only the deletion. Committing leaves a CLEAN tree so the merge succeeds AND the move is durable (no checkout can resurrect the old copy -> no duplicate-task-file). Scoped to the two task paths via 'git commit -- pathspec' (partial commit) so it never sweeps a sibling's staged churn or code, mirroring ship.go commitRecord discipline. Best-effort + no-op when .dacli is gitignored, so the --record-branch/generated-repo layout (dacli 193/222, trunk stays code-only) is unaffected; the extra per-close commit lands only in the dogfood repo where .dacli is deliberately tracked and per-close records are already the norm.
