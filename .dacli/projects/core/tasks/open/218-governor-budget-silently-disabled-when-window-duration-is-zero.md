---
id: t-01KZ4WCR7QQ5Y46XZ4ZRMBFP6V
kind: task
created: 2026-08-03T22:37:29Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# Governor budget silently disabled when window duration is zero
## So that
window-tokens with a zero window does not disable the budget it configures
## Acceptance
- [ ] a zero window duration is rejected or defaulted
## Log
