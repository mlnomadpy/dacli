---
id: t-01KZVNJ41JQ2KEDJ96NQP35FWT
kind: task
created: 2026-08-12T19:00:37Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 517
  repo: mlnomadpy/dacli
---
# Fix accept landing detection for squash-merged GitHub pull requests
## So that
verified work merged through dacli's default GitHub squash path can be accepted without a false unlanded override
## Acceptance
- [x] Given a task PR merged by GitHub squash, accept recognizes the PR merge commit on the target branch even though the original task commit is not an ancestor
- [x] The landing check uses the same authoritative PR-first result as dacli pr status before falling back to branch ancestry
- [x] A closed-unmerged PR and an unrelated branch with a similar diff remain unlanded
- [x] Regression coverage reproduces the observed PRs 509, 510, and 511 shape and proves --allow-unlanded is unnecessary after a confirmed squash merge
## Log
- 2026-08-12T19:21:33Z claimed by a-codex-maintainer-djpe71
- 2026-08-12T19:41:48Z accepted by a-root (applied 1 proposal(s))
- 2026-08-12T19:41:48Z verified by `PR #524 merged by GitHub squash; go test -race ./... passed on merged main, including internal/features/acceptance, internal/features/vcs, and internal/gitx` (exit 0) in branch main at 588ac26 — proves that tree builds, not that the work is in trunk
- 2026-08-12T19:41:48Z deliverable: dacli/397-fix-accept-landing-detection-for-squash-merged-github-pull-requests is merged into main
- 2026-08-12T19:41:48Z completed by a-root
- 2026-08-12T19:54:17Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/524 (event 01KZVQ802XBWKRTEKKMZB2FVQQ)
- 2026-08-12T19:54:17Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/524 at merge commit 3ac37a5b9925d9e3c8fa41867176334cba04ece2 into main; local cleanup complete (event 01KZVQS58WHG1YAMYN7DRKSS51)
