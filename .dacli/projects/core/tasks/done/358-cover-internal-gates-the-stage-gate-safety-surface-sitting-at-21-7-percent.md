---
id: t-01KZR80DF45VQZQDXQCVJM3PGR
kind: task
created: 2026-08-11T11:06:02Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Cover internal/gates, the stage-gate safety surface sitting at 21.7 percent
## Acceptance
- [x] every gate check has a test for BOTH outcomes, satisfied and refused, since a gate that only passes is untested
- [x] a gate whose underlying read fails is proven to fail CLOSED, the vacuous-true defect this package already shipped once
- [x] coverage for internal/gates clears the per-package floor added by the same sprint
## Log
- 2026-08-11T11:06:13Z claimed by a-fixer-tnnxsm
- 2026-08-11T11:17:05Z accepted by a-root
- 2026-08-11T11:17:05Z closed WITHOUT verification — no --verify command was given
- 2026-08-11T11:17:05Z deliverable: dacli/358-cover-internal-gates-the-stage-gate-safety-surface-sitting-at-21-7-percent is merged into sprint/15
- 2026-08-11T11:17:05Z completed by a-root
