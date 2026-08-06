---
id: t-01KZB4SQN7XQ467QK5VCQ0BTEJ
kind: task
created: 2026-08-06T08:59:49Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# accept --verify passes while the task's deliverable is absent, recording false completions
## So that
a done task means its deliverable exists, not merely that the tree still builds
## Acceptance
- [ ] accept warns or refuses when the accepted commit/PR for the task is not reachable from trunk at accept time
- [ ] a verify command that names no task-specific check is flagged (lint-level) as too coarse to prove the acceptance criteria
- [ ] a test reproduces the cutvec failure: merge fails, accept --verify 'go build' runs, and the close is refused or loudly warned rather than silent
## Log
