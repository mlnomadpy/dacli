---
id: t-01KZ4WDDG6D62TP3KD69JDXB8A
kind: task
created: 2026-08-03T22:37:50Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 0.3, probable: 0.8, pessimistic: 2}"
---
# catalog wiki publish reports success when git status fails
## So that
a wiki that was never pushed is not reported as up to date
## Acceptance
- [ ] a failed git status is an error not an empty tree
## Log
