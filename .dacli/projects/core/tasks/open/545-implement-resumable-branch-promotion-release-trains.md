---
id: t-01M147R9YKAKFW35P82YSVHZ83
kind: task
created: 2026-08-28T13:08:11Z
created_by: a-root
owner: a-root
github:
  issue: 870
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
depends_on: "[t-01M146BA07Z5BTS3TTB2ADW7D4, t-01M12QX9HEPKAAS1033W6HS45D]"
---
# Implement resumable branch-promotion release trains
## Context
Adopted from GitHub issue #870.

## Parent

Part of #864. This is branch promotion engineering, not authority to create or push a release tag.

## Observed need

Products commonly land task PRs into an integration branch such as `dev`, then promote a bounded accepted set through a release PR to `master`/`main`. Today the operator manually assembles notes, opens and watches the PR, confirms equivalent trees, and cleans branches.

## Objective

Add a resumable release-train transaction for promoting a configured source branch to a protected target branch through GitHub.



## Non-goals

- Tagging or publishing a release.
- Bypassing branch protection.
- Treating source and target branches as separate dacli project/security boundaries.

## Manual workaround today

Operators create the promotion PR and notes, monitor GitHub, merge, fetch, compare trees, reconcile accepted tasks, and prune manually.

## Acceptance
- [ ] Dry-run identifies the exact source/target SHAs, included accepted tasks/PRs, excluded/unverified work, required checks/reviews, and generated release-PR notes without mutation.
- [ ] Apply creates or reuses one canonical promotion PR and persists its identity immediately; reruns resume rather than duplicate it.
- [ ] GitHub unavailable/auth/unknown states fail closed and never become PR-absent or landed.
- [ ] Promotion waits for configured checks/reviews, merges only with explicit project authority, fetches the target, and verifies the merged tree contains the reviewed source tree or reports the exact divergence.
- [ ] Accepted-task and branch cleanup occurs only after freshly observed target landing and preserves unlanded material.
- [ ] The configured branch names are honored exactly; no fallback from `master` to `main` or repository default occurs silently.
- [ ] No command in this feature creates/pushes a `v*` tag or publishes release artifacts without separate explicit authority.
- [ ] Crash/restart fixtures cover interruption after PR create, pending CI, merge, fetch, and cleanup.
## Log
- 2026-08-28T13:32:48Z dependency edit by a-root (event 01M1495C36DJK1445ZYC6SXKXW)
