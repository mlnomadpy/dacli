---
id: d-252-put-the-reclaim-predicate-in-a-shared-store-helper-and-detect-merged-by
kind: note
note_kind: decision
created: 2026-08-04T11:51:08Z
created_by: a-maintainer-cact9c
about: "[[252]]"
---
# 252: put the reclaim predicate in a shared store helper and detect 'merged' by LOCAL ancestry+topology, not a per-slice copy or a gh network probe
## Chose
252: put the reclaim predicate in a shared store helper and detect 'merged' by LOCAL ancestry+topology, not a per-slice copy or a gh network probe
## Rejected
Duplicating the reap loop in both vcs and orchestration (like runGH/prLandStatus are), or classifying 'merged' via checkLanded's gh PR-state call
## Because
store already imports gitx and both slices import store, so a shared store.ReclaimableWorktrees/PruneWorktrees gives ONE safety-checked predicate without an arch_test slice-import violation and without the silent-divergence risk two copies carry. Detecting merged locally (gitx.IsAncestor + TipOnFirstParentMainline, reusing the dacli 168/241 bare-tip guard) keeps the loop's per-cycle sweep cheap and offline — no gh storm across 86 worktrees — and the 'run finished' (task in done/) predicate catches GitHub squash-merges whose commits never become an ancestor of local trunk. Merged worktrees are only force-removed when clean; finished-but-unmerged branches are kept so no un-landed fix is lost (dacli 154/157).
