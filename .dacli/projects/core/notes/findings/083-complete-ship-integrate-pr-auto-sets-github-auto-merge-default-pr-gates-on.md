---
id: f-083-complete-ship-integrate-pr-auto-sets-github-auto-merge-default-pr-gates-on
kind: note
note_kind: finding
created: 2026-07-22T22:30:08Z
created_by: a-8pkc6y4kp7
about: [[083]]
severity: moderate
---
# 083 complete: ship/integrate --pr --auto sets GitHub auto-merge; default --pr gates on gh pr checks
Commit 3b8b21c by a-8pkc6y4kp7. Files: internal/features/vcs/lifecycle.go (prIntegrateTask + new prChecksPass), internal/features/ship/ship.go (prFlags forwards --auto, dry-run plan), their tests, docs/GITHUB.md. AC1: prIntegrateTask --auto runs 'gh pr merge <branch> --auto --merge --delete-branch' and returns landed=false (GitHub merges on CI green; local worktree/branch kept). AC2: without --auto the default --pr path calls prChecksPass -> 'gh pr checks <branch>'; merges only on exit 0 (or 'no checks reported'), else leaves the PR OPEN and reports 'left N PR(s) open' rather than blindly merging. cmdIntegrate now counts merged-now vs open. AC3: go build ./... clean; go test ./internal/... all green; gofmt -l clean. Tests added: TestIntegratePRAutoSetsAutoMerge, TestIntegratePRLeavesOpenWhenChecksNotPassing, TestIntegratePRMergesWhenChecksPass, TestShipForwardsAutoToIntegrate. Box-check refused for non-owner; owner verifies and closes via task check/done + merge --task 083.
