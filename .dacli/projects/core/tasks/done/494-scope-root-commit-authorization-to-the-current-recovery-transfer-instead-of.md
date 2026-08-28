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
- [x] A completed historical worktree transfer to a-root cannot constrain commits in an unrelated later root-owned worktree.
- [x] Claim lookup binds a transfer to its recorded worktree and current branch or task context before using its paths for commit authorization.
- [x] A current audited recovery transfer still constrains root to exactly its declared paths, including after process restart.
- [x] Finished, pruned, or superseded transfer records remain available for attribution but are excluded from unrelated authorization decisions.
- [x] A public-command regression reproduces task 493 being refused by task 492's stale internal/store,internal/features/execution transfer and proves the current exact claim succeeds.
- [x] The refusal names the stale/current transfer provenance when a genuine scope mismatch occurs, without recommending a blind force override.
- [x] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-27T23:43:01Z claimed by a-maintainer-e0s56a
- 2026-08-27T23:44:51Z claimed by a-maintainer-ve10nq
- 2026-08-28T00:13:36Z accepted by a-root
- 2026-08-28T00:13:36Z verified by `env GOCACHE=/tmp/dacli-494-accept-cache go test ./...` (exit 0) in branch dacli/494-scope-root-commit-authorization-to-the-current-recovery-transfer-instead-of at bd65dfd3 — proves that tree builds, not that the work is in trunk
- 2026-08-28T00:13:36Z deliverable: dacli/494-scope-root-commit-authorization-to-the-current-recovery-transfer-instead-of exists but is NOT in main — closed anyway
- 2026-08-28T00:13:36Z completed by a-root
- 2026-08-28T00:15:43Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/843 (event 01M12V0V1Z982K7KJXH7F19T2V)
- 2026-08-28T00:15:43Z a-root: Landing policy override: mode=pr base=main (event 01M12VE7QFSRDEPYXWWMKP7XG7)
- 2026-08-28T00:15:43Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/843 at merge commit f12dae3601d611467f7060ff10c2113e0c6e61e1 into main (generation 0) (event 01M12VEN5WZGY22FRR3QFM638W)
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-494-check-cache go test ./internal/features/vcs ./internal/cli","exit_code":0,"duration_ms":53030,"artifact_hash":"sha256:d9b95faa43696d345016503d7192cc43af26f5ca73c0b1161b804165f35790bf","verifier":"a-root","branch":"dacli/494-scope-root-commit-authorization-to-the-current-recovery-transfer-instead-of","commit_sha":"bd65dfd36dd4d5e8f3b0a6127289eca6d3bd581f"}
{"command":"env GOCACHE=/tmp/dacli-494-accept-cache go test ./...","exit_code":0,"duration_ms":74050,"artifact_hash":"sha256:ecd893936f774dd63f21f2dbd0a92a9d26f6bd916abd3f62120390858aa5767d","verifier":"a-root","branch":"dacli/494-scope-root-commit-authorization-to-the-current-recovery-transfer-instead-of","commit_sha":"bd65dfd36dd4d5e8f3b0a6127289eca6d3bd581f"}
