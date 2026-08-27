---
id: t-01M0N00V8CYZ3S125G5HJ2CTYN
kind: task
created: 2026-08-22T15:04:26Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
github:
  issue: 768
  repo: mlnomadpy/dacli
---
# Scope root commit authorization to the current recovery transfer instead of historical claims
## Acceptance
- [ ] A completed historical worktree transfer to a-root cannot constrain commits in an unrelated later root-owned worktree.
- [ ] Claim lookup binds a transfer to its recorded worktree and current branch or task context before using its paths for commit authorization.
- [ ] A current audited recovery transfer still constrains root to exactly its declared paths, including after process restart.
- [ ] Finished, pruned, or superseded transfer records remain available for attribution but are excluded from unrelated authorization decisions.
- [ ] A public-command regression reproduces task 493 being refused by task 492's stale internal/store,internal/features/execution transfer and proves the current exact claim succeeds.
- [ ] The refusal names the stale/current transfer provenance when a genuine scope mismatch occurs, without recommending a blind force override.
- [ ] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-27T23:43:01Z claimed by a-maintainer-e0s56a
