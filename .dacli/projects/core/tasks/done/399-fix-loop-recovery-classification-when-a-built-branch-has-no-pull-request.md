---
id: t-01KZVRS9E01ZZC2KKR0W1N11E3
kind: task
created: 2026-08-12T19:56:57Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 532
  repo: mlnomadpy/dacli
---
# Fix loop recovery classification when a built branch has no pull request
## Acceptance
- [x] Given a task branch with commits and no GitHub pull request, loop recovery reports the branch as awaiting PR creation rather than closed-unmerged
- [x] Recovery never deletes or clears a local or remote task branch solely because no pull request exists
- [x] A regression reproduces task 366 after run 01KZVR1TQH: verified commit exists, gh reports no PR, and the commit remains reachable after bounded-loop recovery
- [x] Closed-unmerged pull requests are still classified separately and retain the existing safe retry behavior
## Log
- 2026-08-12T20:08:55Z claimed by a-codex-maintainer-csf6ta
- 2026-08-12T20:24:22Z accepted by a-root
- 2026-08-12T20:24:22Z verified by `go test ./internal/features/orchestration -count=1` (exit 0) in branch main at 6fc9e5d — proves that tree builds, not that the work is in trunk
- 2026-08-12T20:24:22Z deliverable: dacli/399-fix-loop-recovery-classification-when-a-built-branch-has-no-pull-request is merged into main
- 2026-08-12T20:24:22Z completed by a-root
- 2026-08-12T20:40:43Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/536 (event 01KZVSRHF2RF1Y56DDNCHJCGPX)
- 2026-08-12T20:40:43Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/536 at merge commit 32df6d73ad032082150dbda41aa96a8c193e5390 into main; local cleanup complete (event 01KZVT3YD7XJP84V2A4BP9RJ60)
