---
id: t-01KZP7AJSNSAMTA3MXCQ0KJJ4F
kind: task
created: 2026-08-10T16:15:37Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# ship --push leaves the workspace record local, so the trajectory never reaches the remote
## Acceptance
- [ ] ship --push pushes the record branch as well as the current branch when they differ
- [ ] the push line names every ref actually pushed, so 'pushed main' cannot stand in for an unpushed record
- [ ] a test asserts the record ref advances on the remote after ship --push with record_branch set
## Log
