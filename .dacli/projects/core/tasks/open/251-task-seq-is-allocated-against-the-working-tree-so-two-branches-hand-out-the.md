---
id: t-01KZ65ZG4XYFKTKT5YMH4HWNAG
kind: task
created: 2026-08-04T10:44:15Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 8}"
---
# Task seq is allocated against the working tree so two branches hand out the same number
## So that
a merged branch cannot produce two different tasks that share a reference
## Acceptance
- [ ] filing a task on a branch cannot reuse a seq already taken on another unmerged branch
- [ ] a regression test reproduces the collision across two branches
- [ ] existing duplicate seqs are reconciled rather than silently left
## Log
