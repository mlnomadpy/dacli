---
id: t-01KZVND7HYHCSBSBSFF5TTXS6G
kind: task
created: 2026-08-12T18:57:56Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 516
  repo: mlnomadpy/dacli
---
# Fix integrate reporting failure after GitHub merged but local worktree blocks branch deletion
## So that
operators can trust the integration result and recover automatically when the remote merge succeeds before local branch cleanup
## Acceptance
- [x] A regression keeps a task branch attached to a worktree, merges its PR successfully, and proves integrate reports the remote landing rather than zero integrated branches
- [x] Post-merge local branch deletion failure is classified as cleanup debt, not merge failure, and the task receives a durable integration event with the merge commit
- [x] The cleanup path removes or schedules removal of the finished worktree before deleting the local branch without risking uncommitted work
- [x] Re-running integrate after this partial state is idempotent and does not attempt to merge the already-merged PR again
## Log
- 2026-08-12T19:02:41Z claimed by a-codex-maintainer-2pkfj7
- 2026-08-12T19:19:34Z accepted by a-root (applied 1 proposal(s))
- 2026-08-12T19:19:34Z verified by `GOCACHE=/private/tmp/dacli-gocache GOTMPDIR=/private/tmp go test ./internal/features/vcs` (exit 0) in branch main at ed41cb8 — proves that tree builds, not that the work is in trunk
- 2026-08-12T19:19:34Z deliverable: dacli/396-fix-integrate-reporting-failure-after-github-merged-but-local-worktree-blocks exists but is NOT in main — closed anyway
- 2026-08-12T19:19:34Z completed by a-root
- 2026-08-12T19:29:26Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/519 (event 01KZVP4CPFQE1VJQQ7Z9S0QDQ6)
