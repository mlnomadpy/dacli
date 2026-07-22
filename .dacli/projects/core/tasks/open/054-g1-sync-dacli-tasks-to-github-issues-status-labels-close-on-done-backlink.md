---
id: t-01KY5A69PFBM1XB8EQSV0JFK9B
kind: task
created: 2026-07-22T16:22:55Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# G1: sync dacli tasks to GitHub Issues (status labels, close on done, backlink)
## Acceptance
- [ ] on ship/sync, each dacli task materializes as a GitHub Issue: title/body from the task, a status label mirroring the folder, closed when the task is done; the issue URL is recorded back on the task (idempotent, no duplicates)
- [ ] opt-in and rate-aware: syncs in a batch off the agent hot path, never per-spawn; a private repo by default
- [ ] committed by an agent; build + test green
## Log
