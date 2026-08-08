---
id: f-161-added-dacli-pr-status-gh-pr-state-merged-open-closed-automergerequest
kind: note
note_kind: finding
created: 2026-07-26T22:23:01Z
created_by: a-4k8g38rpse
about: [[161]]
severity: moderate
---
# 161: added dacli pr status — gh PR state (merged/OPEN/closed+autoMergeRequest) checked before any local branch-vs-main compare, closing the 157/160 false-positive class
internal/features/vcs/lifecycle.go: checkLanded()/cmdPRStatus, new command 'dacli pr status <task>'. gh pr list --head <branch> --state all --json state,url,autoMergeRequest decides merged/landing/orphaned; only when gh reports no PR at all does it fall back to gitx.RunNetwork fetch origin <into> + gitx.IsAncestor(branch, origin/<into>) (internal/gitx/gitx.go new IsAncestor helper) — never a stale local branch-vs-current-main. Tests: internal/features/vcs/prstatus_test.go (7 cases incl. the exact 157/160 shape: OPEN PR with autoMergeRequest queued -> landing, not orphaned) and internal/gitx/gitx_test.go (3 IsAncestor cases). Docs: docs/GITHUB.md new section 9.6 names 157/160 explicitly and states the rule going forward. internal/prompts/tpl/review_workflow.md now tells reviewer-role agents to run 'dacli pr status' instead of a bare git merge-base check before calling a branch orphaned.
