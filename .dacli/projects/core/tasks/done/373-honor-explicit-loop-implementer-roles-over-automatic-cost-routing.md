---
id: t-01KZV2X46ATDJJNSJKB8FZWCY3
kind: task
created: 2026-08-12T13:34:34Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
depends_on: [370]
github:
  issue: 466
  repo: mlnomadpy/dacli
---
# Honor explicit loop implementer roles over automatic cost routing
## So that
An operator-supplied --impl-role selects the requested runtime and method predictably
## Acceptance
- [x] loop --impl-role R spawns every build task with R unless a phase gate explicitly refuses that role
- [x] Cheapest-capable per-task routing remains the default only when --impl-role is omitted
- [x] Dry-run output and loop advise explain whether the role came from an explicit override, phase routing, or automatic cost routing
- [x] A regression roster proves an explicit backend role is not replaced by a cheaper frontend role for a task mentioning docs/RUNTIMES.md
- [x] go test -race ./... passes
## Log
- 2026-08-12T19:21:39Z claimed by a-codex-maintainer-hyzqzv
- 2026-08-12T19:41:59Z accepted by a-root
- 2026-08-12T19:41:59Z verified by `PR #526 merged; go test -race ./... passed on merged main, including orchestration explicit-role routing regressions` (exit 0) in branch main at 588ac26 — proves that tree builds, not that the work is in trunk
- 2026-08-12T19:41:59Z deliverable: dacli/373-honor-explicit-loop-implementer-roles-over-automatic-cost-routing is merged into main
- 2026-08-12T19:41:59Z completed by a-root
- 2026-08-12T19:54:17Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/526 (event 01KZVQ8BVE6TEFETJ59YACFQ43)
- 2026-08-12T19:54:17Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/526 at merge commit 588ac260c97fb16abb0ce923add828545792a19e into main; local cleanup complete (event 01KZVQSS3TXQ5YVE5XP9AZ0VT2)
