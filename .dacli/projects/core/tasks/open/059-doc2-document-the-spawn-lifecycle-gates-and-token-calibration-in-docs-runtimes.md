---
id: t-01KY5G3H3G8WCYJ2RSMSKDW3MV
kind: task
created: 2026-07-22T18:06:16Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# DOC2: document the spawn lifecycle, gates, and token calibration in docs/RUNTIMES.md
## Acceptance
- [ ] docs/RUNTIMES.md documents the spawn flags actually present (--advise, --claim, --detach, --worktree, --max-tokens, --force, --review, --pr) and the lifecycle commands (agents/--tail, kill, wait, logs, accept, ship, integrate) — verified against internal/features/execution and vcs
- [ ] the token-actuals path is documented: usage_format: stream-json opt-in, usage.txt capture, by-agent (role/model/runtime) calibration bands, the n>=10 provisional gate, and how --advise/--max-tokens use MedianTokenRatio
- [ ] committed by an agent; docs-only
## Log
- 2026-07-22T18:06:28Z claimed by a-5zfa3xx3z5
