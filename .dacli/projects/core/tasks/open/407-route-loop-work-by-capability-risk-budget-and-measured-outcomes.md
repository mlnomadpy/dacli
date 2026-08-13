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
- [ ] Eligibility gates reject models that lack required tools, grant enforcement, context, task capacity, or remaining budget
- [ ] Ranking uses provider-neutral cost tier plus calibrated tokens per completed task and first-pass success with sample counts
- [ ] The loop records a structured routing explanation containing candidates, exclusions, scores, and selected runtime and model
- [ ] Operator runtime or model overrides remain authoritative; automatic routing does not replace either override
- [ ] A rolling provider budget can pause the named provider while eligible alternatives continue through explicit fallback policy
## Log
