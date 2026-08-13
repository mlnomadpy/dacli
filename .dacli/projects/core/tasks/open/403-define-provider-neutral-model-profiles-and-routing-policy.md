---
id: t-01KZX7PXE0HPNVZVAC7DVZ91MC
kind: task
created: 2026-08-13T09:37:02Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
github:
  issue: 547
  repo: mlnomadpy/dacli
---
# Define provider-neutral model profiles and routing policy
## So that
dacli can compare models from Codex, Claude Code, Gemini CLI, Copilot CLI, and future runtimes without vendor-name heuristics
## Context
Use a declarative model catalog plus a pure routing policy. Runtime adapters remain transport concerns; model price, capability, capacity, and context metadata belong to provider-neutral domain objects.
## Acceptance
- [ ] Runtime or role configuration can declare model id, cost tier, maximum task points, context limit, and capability tags without provider-specific fields
- [ ] team assign ranks declared profiles and prints the selected runtime, model, and decision factors
- [ ] Unknown or unpriced models receive tier 99; eligible priced profiles use tiers 1 through 98 and take precedence
- [ ] internal/team contains no hardcoded haiku, sonnet, opus, GPT, Gemini, or Codex cost ordering
- [ ] Migration tests preserve routing for existing role files
## Log
