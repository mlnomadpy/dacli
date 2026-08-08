---
id: t-01KYG6TPKAJ5S2VCWYKR7CD4GA
kind: task
created: 2026-07-26T21:55:48Z
created_by: a-s4764r5zf3
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 6, pessimistic: 10}
---
# loop: fetch + fast-forward local main after async --auto PR merges land, so the next ship --push can't fail non-fast-forward
## So that
task 114's deferred-record-push fix now lets real auto-merges land under strict branch protection, so local main will start falling genuinely behind origin/main across loop cycles and the next plain 'git push -u origin' (no pull/rebase) fails non-fast-forward — task 114 explicitly scoped this reconciliation out as a separate, larger change
## Acceptance
- [x] Add a gitx sync helper (git fetch origin + merge --ff-only origin/<default-branch>) in internal/gitx/gitx.go with a unit test — no such Pull/Fetch/Sync/FastForward function exists today (only Push at gitx.go:219)
- [x] The orchestration loop driver (internal/features/orchestration/orchestration.go) fetches + fast-forwards local main between cycles once fixer PRs have merged asynchronously via 'gh pr merge --auto' (lifecycle.go:757-767 returns immediately with no later sync; the only pull --ff-only at lifecycle.go:815 fires ONLY on the synchronous gated gh-merge path, never on --auto)
- [x] ship --push (internal/features/ship/ship.go:168) and dacli push retry after a fetch+ff (or fetch+rebase) on a non-fast-forward rejection instead of surfacing a bare 'push failed: <out>'
- [x] The record-step push error the orchestration driver currently swallows is surfaced; a regression test proves a local main diverged behind origin (after an --auto merge) is reconciled rather than stranding the next push
- [x] go build ./... clean and go test ./internal/... green
## Log
- 2026-07-26T21:59:22Z claimed by a-kbvnat1ma2
- 2026-07-26T22:06:08Z adopted by a-root (owner a-s4764r5zf3 orphaned)
- 2026-07-26T22:06:08Z accepted by a-root
- 2026-07-26T22:06:08Z completed by a-root
- 2026-08-03T22:38:15Z a-kbvnat1ma2: PR opened: https://github.com/mlnomadpy/dacli/pull/268 (event 01KYG7DCEWXK0SEPZK9A6X4SBG)
