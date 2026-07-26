---
id: t-01KYG7N4Y930QTT71MFA82P4HV
kind: task
created: 2026-07-26T22:10:15Z
created_by: a-y7ksaqj45b
owner: a-root
priority: should
---
# Re-integrate task 159's fetch+ff / push-retry fix: gitx sync helper + loop trunk sync + ship --push retry never landed on main (accepted+closed but branch orphaned)
## Acceptance
- [x] Re-integrate branch dacli/159-... (commit 8544493, +315 across 6 files) onto main: gitx.FastForward + PushSync exist in internal/gitx/gitx.go — verify with 'git merge-base --is-ancestor 8544493 main' (today: NOT ON MAIN; Push at gitx.go:219 is still a bare 'push -u origin <branch>' with no helper)
- [x] ship --push (internal/features/ship/ship.go:168) and dacli push retry after a fetch+ff on a non-fast-forward rejection instead of surfacing a bare 'push failed: <out>' (today main still returns the bare error)
- [x] The orchestration loop driver (internal/features/orchestration/orchestration.go) fetches + fast-forwards local main between cycles after async 'gh pr merge --auto' merges land, reconciling divergence rather than stranding the next push (today main's only ff reconciliation is vcs/lifecycle.go:815 'pull --ff-only', which fires solely on the synchronous gated path, never on --auto)
- [x] go build ./... clean and go test ./internal/... green, including 8544493's gitx_test.go (146 lines) and orchestration/driver_test.go regression that proves a local main diverged behind origin after an --auto merge is reconciled
- [x] Root-cause spot-check: confirm whether the loop's accept-close-without-merge (the same mechanism behind [[115]] and [[157]]) orphaned 159, and re-run the merge-base orphan sweep across recent done tasks; if the loop mechanism is at fault link [[115]] rather than re-fixing the loop here
## Log
- 2026-07-26T22:13:09Z adopted by a-root (owner a-y7ksaqj45b orphaned)
- 2026-07-26T22:13:09Z accepted by a-root
- 2026-07-26T22:13:09Z completed by a-root
