---
id: t-01M0ZCAQ05J2H9VHB4BA9YTQGD
kind: task
created: 2026-08-26T15:51:56Z
created_by: a-root
owner: a-root
github:
  issue: 791
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] accept ignores configured landing base after a confirmed merge
## Context
Adopted from GitHub issue #791.

Observed with landing.mode=pr and landing.base=dev: after the task pull request was confirmed merged into dev, dacli accept still evaluated the task commit against master and refused closure as unlanded. Manual workaround: verify the merged PR and fresh dev ancestry, then use --allow-unlanded. Related but distinct from #790, which covers PR creation choosing a default base. Expected: acceptance resolves the effective landing base and authoritative merged-PR state, so a PR merged into configured dev is accepted without an override. Acceptance criteria: add a regression with repository default master, configured landing base dev, and a confirmed merge into dev; prove accept succeeds and never treats master as the target. Non-goal: changing the repository default branch.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] Acceptance resolves the project’s effective landing base rather than assuming `main` or the repository default branch.
- [ ] A regression with repository default `master`, configured landing base `dev`, and a confirmed PR merge into `dev` accepts without `--allow-unlanded`.
- [ ] The same regression proves acceptance never evaluates the task commit against `master` when `dev` is effective.
- [ ] Remote lookup or ancestry failures fail closed with the selected base named in the diagnostic.
- [ ] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-26T23:13:39Z claimed by a-root
