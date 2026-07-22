---
id: d-auto-uses-gh-pr-merge-auto-merge-default-pr-gates-on-gh-pr-checks
kind: note
note_kind: decision
created: 2026-07-22T22:29:16Z
created_by: a-8pkc6y4kp7
about: [[083]]
---
# auto uses gh pr merge --auto --merge; default --pr gates on gh pr checks
## Chose
auto uses gh pr merge --auto --merge; default --pr gates on gh pr checks
## Rejected
keep blind gh pr merge and add --auto as a separate always-merge flag
## Because
acceptance needs hands-off integration: --auto queues GitHub native auto-merge (merge on CI green) so operator never waits; without --auto ship merges only PRs whose gh pr checks pass and leaves red/pending open instead of blindly merging. Logic in prIntegrateTask (internal/features/vcs/lifecycle.go); ship forwards --auto via prFlags. prIntegrateTask now returns (landed bool, err) so cmdIntegrate counts merged-now vs left-open. --auto keeps local worktree/branch since GitHub owns the pending merge; offline --auto/--no-merge surface the error not silently local-merge
