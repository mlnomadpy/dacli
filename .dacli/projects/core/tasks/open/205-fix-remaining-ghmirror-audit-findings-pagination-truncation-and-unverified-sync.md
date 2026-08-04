---
id: t-01KZ4W93AAMGPH7ETEFBNNAAMS
kind: task
created: 2026-08-03T22:35:29Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Fix remaining ghmirror audit findings: pagination truncation and unverified sync counts
## So that
a mature repo does not silently mirror only its first 1000 issues
## Acceptance
- [ ] list calls detect a hit limit
- [ ] synced counts only increment on verified writes
## Log
