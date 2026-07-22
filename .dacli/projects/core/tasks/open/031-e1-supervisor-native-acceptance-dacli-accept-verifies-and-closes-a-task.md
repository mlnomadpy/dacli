---
id: t-01KY53QHFJ381DVNKHSHPFFJ56
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# E1: supervisor-native acceptance — dacli accept verifies and closes a task
## Acceptance
- [ ] dacli accept <task> verifies the agent's completion (build/test hook + acceptance criteria) and applies box-checks + done in one step, so the owner sets policy instead of hand-closing every spawn
- [ ] an agent can PROPOSE box-checks as events that dacli sync/accept applies (owner still owns the decision), removing the per-task manual close
- [ ] committed on branch by an agent; build + test green
## Log
