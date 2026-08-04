---
id: t-01KZ702QCG6K5RYHGJ6WXVCY0N
kind: task
created: 2026-08-04T18:20:23Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# task done closes a task whose Acceptance section is empty, so zero boxes counts as all boxes
## So that
a closed task means something was verified rather than that nothing was ever asked for
## Acceptance
- [ ] closing a task with no acceptance criteria is refused or explicitly marked unverified
- [ ] the rule is the same on every close path: task done, accept, and the propose-then-sync route
## Log
