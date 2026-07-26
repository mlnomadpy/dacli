---
id: f-loop-s-local-main-checkout-never-syncs-with-origin-after-an-async-gh-pr-merge
kind: note
note_kind: finding
created: 2026-07-26T21:52:13Z
created_by: a-hp40s2ycrb
about: [[114]]
severity: moderate
---
# loop's local main checkout never syncs with origin after an async gh pr merge --auto lands (separate latent bug)
internal/features/orchestration/orchestration.go's driver never fetches+fast-forwards local main after a fixer's PR merges asynchronously via gh pr merge --auto. Confirmed: no Pull/FastForward/Sync helper exists in internal/gitx/gitx.go, cmdWait (execution.go) never touches git, gitx.AddWorktree branches off local HEAD with no prior fetch, and cmdPR's --auto path (lifecycle.go:211-218) returns immediately without ever syncing local main later. The ONLY existing fast-forward (lifecycle.go:815 git pull --ff-only) fires solely on the synchronous non-auto gh pr merge path, never on --auto. Once task 114's fix (deferred record push, orchestration.go recordSelfPR/stillPending) unblocks real auto-merges under strict protection, local main will start falling genuinely behind origin/main across cycles; the next ship --push (internal/features/ship/ship.go:166-172, a plain git push -u origin <branch> with no pull/rebase retry) will then fail non-fast-forward -- and the error is currently swallowed (orchestration.go's d.run.run("record", ...) call discarded (out, err) before my fix; my fix at least logs it now). Not fixed here: fixing it correctly requires reconciling local-only accumulated .dacli record commits against upstream merge commits that may touch the same task-store files (the fixer's own branch also edits .dacli/projects/.../tasks/*.md), which is a materially larger, separate change than task 114's scope (stopping the mid-cycle push from stranding in-flight PRs). Worth its own task.
