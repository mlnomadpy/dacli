---
id: t-01KZXN0N2E9B96Z8610V8J9E35
kind: task
created: 2026-08-13T13:29:33Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 572
  repo: mlnomadpy/dacli
---
# Document GitHub collaboration App boundary Docker and swarm operations
## Acceptance
- [x] docs/GITHUB.md clearly separates shipped gh-based behavior from proposed GitHub App behavior.
- [x] The guide explains when no App is needed and when an installation-scoped App is valuable, including least-privilege and event trust boundaries.
- [x] Docker is explained as an execution envelope and linked to its existing issue without being claimed as shipped.
- [x] An end-to-end GitHub-first swarm workflow covers issue mapping, claims, worktrees, PRs, CI wait, dacli integration, closure, and record push.
- [x] Documentation support tests and go test ./... pass.
## Log
- 2026-08-13T13:30:40Z claimed by a-fixer-devvfk
- 2026-08-13T14:04:24Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T14:04:24Z verified by `GOCACHE=/private/tmp/dacli-docs-416-accept go test ./docs` (exit 0) in branch main at 73aecf2 — proves that tree builds, not that the work is in trunk
- 2026-08-13T14:04:24Z completed by a-root
- 2026-08-13T14:13:09Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/576 (event 01KZXQ1BVKTH38XQ77540777PQ)
- 2026-08-13T14:13:09Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/576 at merge commit 8d8edf07f358116e9b5715885b65005347ddd90f into main; local cleanup complete (event 01KZXQDF9A53S422WH5S0PH85E)
