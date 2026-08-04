---
id: t-01KZ6S4CDF9964NZVC9NBEF9MW
kind: task
created: 2026-08-04T16:18:58Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 6}"
---
# github push mirrors the whole backlog or nothing, so a mature project cannot mirror one wave
## So that
an operator with a long backlog can file the tasks that matter without creating a hundred issues, and never has to reach for raw gh
## Acceptance
- [ ] push accepts a task window (explicit refs and/or a since window) and mirrors only those
- [ ] an issue that already exists for a task is adopted into the mapping rather than duplicated, even when the issue body carries no dacli marker
- [ ] a test covers a workspace whose done set is far larger than the window
## Log
- 2026-08-04T16:19:51Z claimed by a-maintainer-df2nne
