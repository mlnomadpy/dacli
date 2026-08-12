---
id: t-01KZV5RSNF2C50DHC8HGSPJMNX
kind: task
created: 2026-08-12T14:24:38Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 470
  repo: mlnomadpy/dacli
---
# Make loop worker timeout configurable and estimate-aware
## Acceptance
- [x] loop accepts an explicit worker timeout and passes it to implementation and review spawns instead of always recording timeout_s: 300
- [x] when the flag is omitted, the default timeout is derived from a documented task-estimate or role policy and a task with Te above the five-minute band is not killed at 300 seconds
- [x] driver tests assert the exact spawn arguments for explicit and derived timeouts, and go test ./internal/features/orchestration/... passes
## Log
- 2026-08-12T15:35:24Z claimed by a-codex-maintainer-vxzmpg
- 2026-08-12T15:41:00Z completed by a-root
