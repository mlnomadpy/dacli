---
id: t-01KZPDWTRP1SF1BPWB50022Y3F
kind: task
created: 2026-08-10T18:10:27Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# github push creates decision issues but never closes them, so records accumulate as open work forever
## Acceptance
- [ ] a decision note mirrored as an issue is either filed closed, or closed on the next push, so the issue list holds work rather than records
- [ ] the behaviour is chosen deliberately and stated in the command's brief, not left implicit
- [ ] a test asserts a decision issue does not remain open after a second push
## Log
