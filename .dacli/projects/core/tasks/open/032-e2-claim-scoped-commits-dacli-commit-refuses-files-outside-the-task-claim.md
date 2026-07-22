---
id: t-01KY53QHG2G4WGQ4XNF3XT8D23
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# E2: claim-scoped commits — dacli commit refuses files outside the task claim
## Acceptance
- [ ] dacli commit warns or refuses when staged files fall outside the spawn's declared --claim scope (plus the task file), killing the git-add-A staging-boilerplate class
- [ ] the spawn's claim is recorded where commit can read it; --force overrides with a loud note
- [ ] committed on branch by an agent; build + test green
## Log
