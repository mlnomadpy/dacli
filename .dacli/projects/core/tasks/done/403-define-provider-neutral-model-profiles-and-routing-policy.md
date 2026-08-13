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
- [x] Runtime or role configuration can declare model id, cost tier, maximum task points, context limit, and capability tags without provider-specific fields
- [x] team assign ranks declared profiles and prints the selected runtime, model, and decision factors
- [x] Unknown or unpriced models receive tier 99; eligible priced profiles use tiers 1 through 98 and take precedence
- [x] internal/team contains no hardcoded haiku, sonnet, opus, GPT, Gemini, or Codex cost ordering
- [x] Migration tests preserve routing for existing role files
## Log
- 2026-08-13T09:44:20Z claimed by a-codex-maintainer-j9h9mq
- 2026-08-13T10:11:22Z accepted by a-root
- 2026-08-13T10:11:22Z verified by `cd /Users/tahabsn/Documents/GitHub/dacli/.dacli/worktrees/core-403-define-provider-neutral-model-profiles-and-routing-policy && GOCACHE=/private/tmp/dacli-403-gocache go test ./internal/team ./internal/store ./internal/features/teamops` (exit 0) in branch main at f244e11 — proves that tree builds, not that the work is in trunk
- 2026-08-13T10:11:22Z completed by a-root
- 2026-08-13T10:47:41Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/559 (event 01KZX98DXBYQGK02R3VPMCSHYD)
- 2026-08-13T10:47:41Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/559 at merge commit ad87352325a873f0c4e1d596093cf1b6649676e5 into main; local cleanup complete (event 01KZX9PESR0C58GR5ME6HEQCWQ)
