---
id: t-01KZYW7MC5V9B7QBMXFMAVT5VG
kind: task
created: 2026-08-14T00:54:56Z
created_by: a-root
owner: a-root
priority: must
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

## Live post-merge reproduction (2026-08-16)

PR #678 auto-merged task 452 and GitHub deleted its remote task branch before `dacli integrate` had written an `Integrated via PR` event. From a clean, current `main`, `dacli integrate --pr --tasks 452 --into main` pushed the stale attached local branch back to GitHub, then failed `gh pr create` with `No commits between main and dacli/452-...`. The recorded-landing cleanup retry added by PR #678 was therefore unreachable on this normal auto-merge path.

Implementation and claim boundary: `internal/features/vcs` owns remote integration and its end-to-end regressions; `internal/gitx` owns the shared linked-worktree and branch cleanup primitive.

## Acceptance criteria

- [ ] A regression fixture creates a task branch checked out in a dacli-managed linked worktree and a mocked/fixture PR that becomes merged.
- [ ] After the remote merge is confirmed, integration records one integrated task even when local cleanup initially encounters the attached worktree.
- [ ] The integration path removes or prunes the linked worktree before deleting its checked-out branch, using the shared cleanup primitive rather than command-specific filesystem deletion.
- [ ] Successful cleanup leaves no registered task worktree and no stale local task branch.
- [ ] A genuine cleanup failure reports the merged landing and a specific recoverable cleanup warning/error without claiming zero integrations or reverting task landing state.
- [ ] Re-running integration or `worktree prune` after an interrupted cleanup is idempotent and does not duplicate landing events.
- [ ] Before pushing or creating a PR, integration discovers an already-merged PR for the local task branch even when GitHub deleted the remote branch, records its merge identity, and does not recreate the remote branch.
- [ ] A live-shaped regression begins with `main` containing the task commit, an attached local task worktree/branch, no remote task ref, and no prior landing event; one integration run reports the existing landing and completes cleanup without calling PR creation.
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
- [ ] Before pushing or creating a PR, integration discovers an already-merged PR for the local task branch even when GitHub deleted the remote branch, records its merge identity, and does not recreate the remote branch.
- [ ] A live-shaped regression begins with `main` containing the task commit, an attached local task worktree/branch, no remote task ref, and no prior landing event; one integration run reports the existing landing and completes cleanup without calling PR creation.
- [ ] Mutation evidence proves the regression fails when branch deletion again precedes worktree cleanup.
- [ ] Focused VCS/worktree tests and `go test ./...` pass.
## Log
- 2026-08-16T18:01:47Z claimed by a-maintainer-6w1mv4
- 2026-08-16T18:30:49Z accepted by a-root
- 2026-08-16T18:30:49Z verified by `GOCACHE=/tmp/dacli-452-final go test ./...` (exit 0) in branch dacli/418-continuous-improvement-file-the-single-highest-value-evidence-based-change at 27acc22 — proves that tree builds, not that the work is in trunk
- 2026-08-16T18:30:49Z deliverable: dacli/452-agent-report-integrate-reports-failure-after-remote-pr-merge-when-task-branch is merged into main
- 2026-08-16T18:30:49Z completed by a-root
- 2026-08-16T18:32:25Z reopened by a-root: Live post-merge proof found the auto-merged/deleted-remote-branch path still pushes the stale local branch and fails PR creation before landing recovery can run (cleared 8 acceptance box(es) — the close claimed work that was not verified)
- 2026-08-16T18:38:50Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/678 (event 01M05W3M13G0XZ1KPZ2S37N84M)
- 2026-08-16T18:38:50Z a-root: Landing policy override: mode=pr base=main (event 01M05XGBT3GYCXDPGV4V7436KF)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-452-accept go test ./...","exit_code":0,"duration_ms":72041,"artifact_hash":"sha256:a931ab77971610bf90c6b831c1309b29f03e7ddcf6110826380bb8da29b3f18a","verifier":"a-root","branch":"dacli/418-continuous-improvement-file-the-single-highest-value-evidence-based-change","commit_sha":"27acc22f7d7fd5d7e408acecae81b4a2b7826ada"}
{"command":"GOCACHE=/tmp/dacli-452-final go test ./...","exit_code":0,"duration_ms":69116,"artifact_hash":"sha256:8abae13dccc8090d096a14a776d3a407f5c24a0906d09c32a8f11c59ee730169","verifier":"a-root","branch":"dacli/418-continuous-improvement-file-the-single-highest-value-evidence-based-change","commit_sha":"27acc22f7d7fd5d7e408acecae81b4a2b7826ada"}
