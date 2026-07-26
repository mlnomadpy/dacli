---
id: 01KYG6VCXKVCDB4JR49V9YS41K
kind: event
event_kind: finding
created: 2026-07-26T21:56:11Z
created_by: a-s4764r5zf3
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
Verified in source: loop never fetches/ff local main after --auto PR merges; next push fails non-fast-forward (filed task 159)

Confirmed by reading source (not just the sibling lead f-loop-s-local-main-checkout-never-syncs): (1) internal/gitx/gitx.go exposes Run, RunNetwork, CurrentBranch, IsClean, BranchExists, AddWorktree, ListWorktrees, RemoveWorktree, Merge, Push — NO Pull/Fetch/Sync/FastForward. (2) gitx.Push (gitx.go:219-221) = RunNetwork(root,'push','-u','origin',branch), no pull/rebase retry; ship --push calls it at ship.go:168 and surfaces only 'push failed: <out>'. (3) In prIntegrateTask (internal/features/vcs/lifecycle.go), the --auto path (lines 757-767) runs 'gh pr merge --auto --merge --delete-branch' and returns false,nil immediately — GitHub owns the async merge, local main is never synced afterward. (4) The ONLY 'git pull --ff-only' (lifecycle.go:815) sits in the synchronous gated 3c path AFTER a local gh merge, never reached under --auto. (5) gitx.AddWorktree (gitx.go:138-150) branches off local HEAD with no prior fetch, so each new cycle's worktree also forks from a stale base. Net: once task 114's deferred-record-push fix lets auto-merges actually land under strict branch protection, local main falls progressively behind origin/main across loop cycles and the next 'dacli ship --push' fails non-fast-forward. Task 114's own author scoped this reconciliation out as a separate, larger change. This is a lead for task 159, verify before treating as fact.
