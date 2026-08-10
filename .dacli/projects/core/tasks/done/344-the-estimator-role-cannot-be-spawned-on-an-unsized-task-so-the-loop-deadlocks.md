---
id: t-01KZPR5DTK97V4TDMWM7XK2Y32
kind: task
created: 2026-08-10T21:09:54Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# The estimator role cannot be spawned on an unsized task, so the loop deadlocks on everything review files
## Acceptance
- [x] a capped planner-kind role can be spawned on a task with no estimate, because producing that estimate is its output
- [x] every other capped kind still refuses unsized work, and a planner over its cap is still refused once a size exists
- [x] a cycle whose tasks remain unsized after the sizing step says so and names them, rather than leaving the cause under a 'no net progress' halt
- [x] the refusal message states that a planner-kind role is exempt, so the next reader learns the rule from the error
## Log
- 2026-08-10T21:10:06Z accepted by a-root
- 2026-08-10T21:10:06Z verified by `go test ./internal/features/execution/ ./internal/features/orchestration/` (exit 0)
- 2026-08-10T21:10:06Z deliverable: no dacli/344-the-estimator-role-cannot-be-spawned-on-an-unsized-task-so-the-loop-deadlocks branch — nothing to check against sprint/6
- 2026-08-10T21:10:06Z completed by a-root
