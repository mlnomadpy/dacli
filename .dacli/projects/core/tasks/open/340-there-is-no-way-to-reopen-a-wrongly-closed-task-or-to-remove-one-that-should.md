---
id: t-01KZPG7MNVWCZQQ11ZM3KBQJKC
kind: task
created: 2026-08-10T18:51:18Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# There is no way to reopen a wrongly-closed task, or to remove one that should never have existed
## Acceptance
- [ ] a closed task can be reopened through a command, with the reopen and its reason recorded in the task log
- [ ] a task can be removed, refusing if anything references it, so a probe or duplicate does not have to be deleted by hand
- [ ] neither command silently unchecks acceptance boxes: the boxes it clears and why are stated in the output
## Log
