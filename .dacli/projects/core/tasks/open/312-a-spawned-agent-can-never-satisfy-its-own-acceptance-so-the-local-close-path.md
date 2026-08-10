---
id: t-01KZNX4RDQSH7CSBB1RB7PD450
kind: task
created: 2026-08-10T13:17:41Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# a spawned agent can never satisfy its own acceptance, so the local close path deadlocks
## So that
an unattended loop with no remote can actually close the work its agents finish, instead of thrashing until the guard halts it
## Acceptance
- [ ] a loop cycle in a no-remote workspace closes a task whose agent committed work and proposed done, without an operator running accept by hand
- [ ] the fix names which side gives: either sync verifies acceptance as the owner it already acts as, or the agent protocol proposes through the channel accept --all consumes
- [ ] a test drives the full agent lifecycle (claim, work, propose) and asserts the task reaches done
## Log
