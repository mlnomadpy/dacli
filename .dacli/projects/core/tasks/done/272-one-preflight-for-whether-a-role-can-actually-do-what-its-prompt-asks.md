---
id: t-01KZ6RVF7Z3CEW0FMAVV8MCB9F
kind: task
created: 2026-08-04T16:14:06Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# One preflight for whether a role can actually do what its prompt asks
## So that
a capability mismatch is caught before a run burns rather than after it
## Acceptance
- [x] a single command checks grant against runtime write capability, the dacli binary path against the allowlist, and the role prompt's named tools against what the runtime permits
- [x] it reports every mismatch in one pass rather than failing on the first
- [x] spawn runs it and refuses or warns per the existing convention for each class
## Log
- 2026-08-05T13:35:31Z claimed by a-fixer-p5ee58
- 2026-08-05T14:01:06Z accepted by a-root
- 2026-08-05T14:01:06Z closed WITHOUT verification — no --verify command was given
- 2026-08-05T14:01:06Z completed by a-root
