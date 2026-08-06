---
id: f-273-complete-on-branch-dacli-273-closetask-commits-its-own-record-move
kind: note
note_kind: finding
created: 2026-08-04T20:28:55Z
created_by: a-maintainer-yd9g1p
about: "[[273]]"
severity: moderate
---
# 273 complete on branch dacli/273-...: CloseTask commits its own record move
Commit 61ea12b (a-maintainer-yd9g1p) on branch dacli/273-accept-moves-the-task-file-but-leaves-the-move-uncommitted-so-doctor-reports-it. FIX: store.CloseTask now COMMITS the task-file move after MoveTask (store.go:1195 commitRecordMove) — scoped to the two task paths via 'git commit -- pathspec', best-effort, no-op when git absent or .dacli gitignored (record-branch layout). This closes the dacli-273 duplicate-task-file hazard: a bare os.Rename left the file tracked at its old status path, so a routine checkout/reset resurrected the old copy alongside the new one. Committing (not staging) is required because a STAGED .dacli move makes the 'git merge --no-ff' that integrate/ship runs right after accept REFUSE ('local changes would be overwritten'); a committed move leaves a clean tree so the merge still works. ACCEPTANCE both satisfied: (1) accept (via CloseTask) commits its own record move; (2) tests cover close-then-inspect with no manual git step: store.TestCloseTaskCommitsTheRecordMove + acceptance.TestAcceptCommitsTheRecordMove (real cmdAccept path) + gitx.TestMergeAfterCommittedDacliMove (guards merge invariant). VERIFIED by reproduction: disabling the commit call makes TestCloseTaskCommitsTheRecordMove fail ('open/ copy still tracked at HEAD'); restoring it passes. go build/vet clean, gofmt clean, full go test ./... green. Owner: verify and close via 'dacli accept 273' (proposal recorded), then merge --task 273.
