---
id: f-audit2-r3-regression-check-prior-audit-fixes-in-vcs-ship-acceptance-gitx-all
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-7sy0x8b84g
about: [[t-01KY5GP5RHAPGVT35FDFS4Z0PB]]
source_event: 01KY5H1JXEF2G8Z0YNHXAC5BDC
github:
  issue: 20
  repo: mlnomadpy/dacli
---
# AUDIT2 R3 regression check: prior audit fixes in vcs/ship/acceptance/gitx all held
Verified by source reading (go build/test blocked by the headless sandbox — arbitrary exec needs approval). CONFIRMED FIXED, no regressions: (1) gitx.Merge non-conflict failure no longer mislabeled as conflict — gitx.go:192-202 returns the real error when diff --diff-filter=U is empty. (2) ship never half-ships on a dirty .dacli — gitx.Merge guards with IsCleanExcept(root,'.dacli') gitx.go:182 so accept's tracked task-file moves don't block the merge; cmdIntegrate distinguishes ExitCode==3 conflict (lifecycle.go:417, exit 0) from a hard error (:422-429, propagated non-zero) so ship stops before commit/push. (3) ship doneRefs emits globally-unique ULIDs (ship.go:272-282), not ambiguous bare seqs. (4) ship record message reports integratedCount(out) actual merges (ship.go:141,287), not len(done). (5) accept stamps 'completed by' via store.CloseTask on BOTH acceptOne (acceptance.go:114) and acceptAll (:153), fixing the E1 calibration-span gap. (6) git/gh subprocesses now carry deadlines — vcs.gitIn 30s (vcs.go:57), cmdPR/postVerdicts 120s (lifecycle.go:148,302), gitx local/network timeouts (gitx.go:20-23). Three NEW defects filed separately (PR-URL-as-finding, cmdPR grant gap, contrib double-count).
