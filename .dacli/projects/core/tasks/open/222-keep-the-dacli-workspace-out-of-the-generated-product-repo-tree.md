---
id: t-01KZ4WDDHW19T5C480BM76PEM5
kind: task
created: 2026-08-03T22:37:51Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Keep the dacli workspace out of the generated product repo tree
## So that
a generated app repo is not 80 percent agent bookkeeping files
## Acceptance
- [ ] dacli new can gitignore the workspace while keeping the record branch
## Log
- 2026-08-04T10:50:16Z claimed by a-maintainer-zmqsrg
