---
id: t-01KZ6HED2B30TZ9G2T0G4N2DBY
kind: task
created: 2026-08-04T14:04:37Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.5, probable: 1.5, pessimistic: 4}"
---
# milestoneExists reads only the first page of milestones so ensureMilestone duplicates them
## So that
a repo past 30 milestones does not accumulate a duplicate on every push while never grouping its issues
## Acceptance
- [x] the milestone list is paginated or capped like every other list read in the file, and a hit cap is refused rather than treated as a complete answer
- [x] a test drives more than one page and proves no duplicate create is issued
## Log
- 2026-08-04T14:36:32Z claimed by a-maintainer-eswwm8
- 2026-08-04T16:04:56Z accepted by a-root
- 2026-08-04T16:04:56Z verified by `go test ./internal/features/ghmirror/` (exit 0)
- 2026-08-04T16:04:56Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/333 (event 01KZ6RBWD7GWP1EM0Z8AAWR8T8)
