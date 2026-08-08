---
id: f-078-complete-on-branch-dacli-078-pr-first-ship-integrate-pr-implemented-all-3
kind: note
note_kind: finding
created: 2026-07-22T21:42:57Z
created_by: a-c79p0msrw8
about: [[078]]
severity: moderate
---
# 078 complete on branch dacli/078-...; PR-first ship/integrate --pr implemented, all 3 acceptance met, build+test green
Commit 135fb57 by a-c79p0msrw8 (maintainer) on branch dacli/078-i1-dacli-ship-pr-push-branches-open-enriched-prs-merge-via-gh-pr-first. Staged 5 files: internal/features/vcs/lifecycle.go, internal/features/vcs/printegrate_test.go, internal/features/ship/ship.go, internal/features/ship/ship_test.go, docs/GITHUB.md.

ACCEPTANCE (all met):
(1) 'ship --pr / integrate --pr push each done branch, open a PR via dacli pr (body from acceptance+findings+Fixes #issue; verify verdicts as review comments), merge via gh pr merge — not a local git merge': cmdIntegrate (lifecycle.go:~468) gains --pr; prIntegrateTask (lifecycle.go) pushes via pushBranch, opens the enriched PR via openPR (reuses prBody + always posts postVerdicts), then merges with 'gh pr merge <branch> --squash --delete-branch' (--merge for a merge commit), tears down worktree/branch, ff-pulls local target. openPR was extracted from cmdPR so the standalone 'dacli pr' and the integrate path share ONE PR builder. ship forwards the flags via prFlags (ship.go).
(2) 'gated + operator-triggered; falls back to local merge with a warning when GitHub unreachable; --no-merge opens PRs and stops': --pr is behind the existing rw grant + is a flag (never automatic). isNetworkErr scans gh/git output; a network failure at push/PR-open warns + falls back to mergeTask (local). A NON-network failure (protected branch/auth/dirty) is surfaced, never mislabeled offline. --no-merge stops after opening PRs ('left open for human review') and, offline, surfaces an error rather than silently local-merging.
(3) 'committed by an agent; build+test green': committed via dacli commit (135fb57). go build ./... clean; go test ./internal/... all green. New tests: printegrate_test.go (5 cases: push+open+merge, --no-merge stops before merge, push-network fallback-to-local-merge, non-network push error surfaced, --no-merge no-offline-fallback) + ship_test.go (2 cases: --pr/--no-merge forwarded, default forwards nothing).

Box-checking refused for non-owner (only a-root). Owner: verify + close via 'dacli accept 078', then 'dacli integrate --tasks 078 --into main' (or 'dacli merge --task 078').
