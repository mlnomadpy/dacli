---
id: t-01KZYW7MC5V9B7QBMXFMAVT5VG
kind: task
created: 2026-08-14T00:54:56Z
created_by: a-root
owner: a-root
github:
  issue: 657
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# [agent-report] integrate reports failure after remote PR merge when task branch is still attached to its dacli worktree
## Context
Adopted from GitHub issue #657.

## Symptom

Running `dacli integrate --tasks 032 --into dev --pr` reused the open PR and merged it remotely, but returned non-zero because Git could not delete the local branch while the dacli-managed worktree still had that branch checked out. `gh pr view` and `dacli pr status` both confirmed the PR was merged. `dacli worktree prune --into dev` then safely reclaimed the worktree.

## Suspected cause

The remote integration cleanup deletes the local branch before removing or pruning the linked dacli worktree. Git correctly refuses to delete a branch that is checked out in another worktree, and the cleanup error is allowed to overwrite the already-proven remote integration outcome.

## Design

Separate the landing verdict from recoverable local cleanup. After GitHub confirms the merge, record the truthful remote landing first, then reclaim the task worktree through the shared worktree-removal primitive before deleting the branch. A cleanup failure must be explicit and recoverable, but must not report zero integrated branches or reverse a confirmed merged outcome.

Implementation and claim boundary: `internal/features/vcs` owns remote integration and its end-to-end regressions; `internal/gitx` owns the shared linked-worktree and branch cleanup primitive.

## Acceptance criteria

- [ ] A regression fixture creates a task branch checked out in a dacli-managed linked worktree and a mocked/fixture PR that becomes merged.
- [ ] After the remote merge is confirmed, integration records one integrated task even when local cleanup initially encounters the attached worktree.
- [ ] The integration path removes or prunes the linked worktree before deleting its checked-out branch, using the shared cleanup primitive rather than command-specific filesystem deletion.
- [ ] Successful cleanup leaves no registered task worktree and no stale local task branch.
- [ ] A genuine cleanup failure reports the merged landing and a specific recoverable cleanup warning/error without claiming zero integrations or reverting task landing state.
- [ ] Re-running integration or `worktree prune` after an interrupted cleanup is idempotent and does not duplicate landing events.
- [ ] Mutation evidence proves the regression fails when branch deletion again precedes worktree cleanup.
- [ ] Focused VCS/worktree tests and `go test ./...` pass.

---
_Originally reported via `dacli report`._

## Acceptance
- [ ] A regression fixture creates a task branch checked out in a dacli-managed linked worktree and a mocked/fixture PR that becomes merged.
- [ ] After the remote merge is confirmed, integration records one integrated task even when local cleanup initially encounters the attached worktree.
- [ ] The integration path removes or prunes the linked worktree before deleting its checked-out branch, using the shared cleanup primitive rather than command-specific filesystem deletion.
- [ ] Successful cleanup leaves no registered task worktree and no stale local task branch.
- [ ] A genuine cleanup failure reports the merged landing and a specific recoverable cleanup warning/error without claiming zero integrations or reverting task landing state.
- [ ] Re-running integration or `worktree prune` after an interrupted cleanup is idempotent and does not duplicate landing events.
- [ ] Mutation evidence proves the regression fails when branch deletion again precedes worktree cleanup.
- [ ] Focused VCS/worktree tests and `go test ./...` pass.
## Log
