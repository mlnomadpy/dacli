---
id: f-252-complete-on-branch-dacli-252-worktree-prune-reclaims-merged-finished
kind: note
note_kind: finding
created: 2026-08-04T11:50:57Z
created_by: a-maintainer-cact9c
about: "[[252]]"
severity: major
---
# 252 complete on branch dacli/252-...: worktree prune reclaims merged/finished checkouts, loop calls it each cycle
Commit 953d852 by a-maintainer-cact9c (staged only the 4 intended files via git add + dacli commit --no-add). BOTH acceptance criteria met: (1) a worktree whose branch is merged OR whose run is finished is pruned — new store.ReclaimableWorktrees/PruneWorktrees (internal/store/worktrees.go) reclaims a worktree when its branch is an ancestor of trunk AND entered via a merge (not a bare-tipped live spawn: reuses gitx.TipOnFirstParentMainline+IsAncestor, the dacli 168/241 guard) with a clean tree, OR when its task file sits in done/. Merged branches are deleted too (like gcBranch); a finished-but-unmerged branch is KEPT so an accepted-yet-unlanded fix is never lost. (2) the prune is an operator command AND the loop calls it — new 'dacli worktree prune [--into <trunk>] [--dry-run]' (internal/features/vcs/lifecycle.go cmdWorktreePrune) plus the loop's between-cycle d.reapWorktrees() (internal/features/orchestration/orchestration.go, called right after reconcilePendingAccepts). Shared predicate in store, so both slices reap identically (no arch_test violation: store is a shared pkg, imports gitx; slices don't import each other). PROOF: go build ./... clean; gofmt -l internal/ clean; go vet ./... clean; go test ./... green with DACLI_AGENT stripped (the lone catalog failure is the known DACLI_AGENT env-leak, passes under 'go test -exec env -u DACLI_AGENT'). New TestWorktreePruneReclaimsMergedAndFinished (internal/cli/lifecycle_test.go) drives all four cases end-to-end (merged→pruned+branch-deleted, done→pruned+branch-kept, live-with-commits→kept, bare-spawn→kept) plus --dry-run; VERIFIED it fails before the change ('no such command: [worktree prune]') by stashing the impl, then passes after. Owner: accept 252 + integrate/merge --task 252.
