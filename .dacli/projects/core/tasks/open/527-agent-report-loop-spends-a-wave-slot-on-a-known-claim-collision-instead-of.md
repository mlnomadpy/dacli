---
id: t-01M12N9W3XJC7JRKHCFGRT0XNG
kind: task
created: 2026-08-27T22:26:29Z
created_by: a-root
owner: a-root
github:
  issue: 838
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] loop spends a wave slot on a known claim collision instead of selecting another ready task
## Context
Adopted from GitHub issue #838.

Reproduction on main 483b32d4: cycle 126 selected critical-path tasks 509 and 510 at width 2 even though both claim internal/features/execution. Execution then reported planned claim collision for 510 and left it open without spawning; the cycle produced no work for that slot. Expected: wave planning must apply the same claim-overlap predicate before finalizing the width-limited set, keep one conflicting task, and backfill with the next eligible non-overlapping ready task or explicitly shrink the wave when none exists. Acceptance: a regression with two zero-slack overlapping tasks and one ready non-overlapping task selects one conflicting task plus the independent task; preview and execution show the same selection; no agent identity or worktree is created for a rejected collision; width and single-harness constraints remain enforced; go test ./... passes. Manual workaround: run width 1 or explicitly spawn only one conflicting task.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
