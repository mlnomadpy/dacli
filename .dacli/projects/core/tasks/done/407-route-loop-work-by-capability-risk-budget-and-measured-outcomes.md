---
id: t-01KZX7PXTE3T89KP45STP6S783
kind: task
created: 2026-08-13T09:37:03Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: "[403:FS, 406:FS]"
github:
  issue: 551
  repo: mlnomadpy/dacli
---
# Route loop work by capability risk budget and measured outcomes
## So that
easy tasks use economical models while risky work receives sufficient capability within operator limits
## Context
Use a composable scoring Strategy over hard gates. Eligibility first: role kind, scope, grant, capabilities, capacity, quota, and budget. Ranking second: cost, calibrated success, token estimate, latency, and domain fit. Emit an explanation object for every selection.
## Acceptance
- [x] Eligibility gates reject models that lack required tools, grant enforcement, context, task capacity, or remaining budget
- [x] Ranking uses provider-neutral cost tier plus calibrated tokens per completed task and first-pass success with sample counts
- [x] The loop records a structured routing explanation containing candidates, exclusions, scores, and selected runtime and model
- [x] Operator runtime or model overrides remain authoritative; automatic routing does not replace either override
- [x] A rolling provider budget can pause the named provider while eligible alternatives continue through explicit fallback policy
## Log
- 2026-08-13T14:21:17Z claimed by a-fixer-ngpzz6
- 2026-08-13T14:33:27Z claimed by a-fixer-ngpzz6 (event 01KZXR011FZH3QMMJG1MTHH846)
- 2026-08-13T14:43:28Z adopted by a-root (owner a-fixer-ngpzz6 orphaned)
- 2026-08-13T14:43:28Z accepted by a-root (applied 1 proposal(s))
- 2026-08-13T14:43:28Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T14:43:28Z deliverable: dacli/407-route-loop-work-by-capability-risk-budget-and-measured-outcomes exists but is NOT in main — closed anyway
- 2026-08-13T14:43:28Z completed by a-root
- 2026-08-13T14:52:02Z accepted by a-root
- 2026-08-13T14:52:02Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T14:52:02Z deliverable: dacli/407-route-loop-work-by-capability-risk-budget-and-measured-outcomes is merged into main
- 2026-08-13T14:52:02Z completed by a-root
- 2026-08-13T15:00:52Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/581 (event 01KZXS90DRSJWHBA0Z7BY307R7)
