---
id: f-gitx-merge-reports-every-merge-failure-as-a-conflict-discarding-the-real-error-a-non-conflict-failure-wrongly-blocks-the-task
kind: note
note_kind: finding
created: 2026-07-21T23:09:25Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
---
# gitx.Merge reports every merge failure as a conflict, discarding the real error — a non-conflict failure wrongly blocks the task
gitx.go:111-122 — on 'git merge --no-ff' failure it runs diff --diff-filter=U, and if no files are conflicted substitutes conflicts=['(merge failed; see git output)'] and returns (conflicts, nil), discarding the original err. A merge that fails for a non-conflict reason (missing branch, unrelated histories, index lock) is misreported to mergeTask (lifecycle.go:196) as a conflict, which then wrongly blocks the task. Propagate the real error when --diff-filter=U yields no conflicted files.
