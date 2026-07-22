---
id: t-01KY5G3H32N3D1JBBBNTZCH0H5
kind: task
created: 2026-07-22T18:06:16Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# DOC1: rebuild README command reference + capabilities overview to match shipped surface
## Acceptance
- [x] README's command reference lists the ACTUAL top-level commands (verify against 'dacli help' and the slice tables) — the 25 currently-absent groups (accept, ship, integrate, kill, logs, wait, calibrate, verify, taint, worktree, supervise, ...) are documented with a one-line purpose each
- [x] a concise 'capabilities' section describes the agent-fleet loop (spawn --claim/--detach -> wait -> accept -> ship), calibration (measure cost, then advise/enforce), the trust/taint gates, and the GitHub mirror — accurate, no invented flags
- [x] committed by an agent; go build ok (docs-only, no code change)
## Log
- 2026-07-22T18:06:28Z claimed by a-49dgq2g3m8
- 2026-07-22T18:15:01Z accepted by a-root
- 2026-07-22T18:15:01Z completed by a-root
