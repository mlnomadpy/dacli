---
id: t-01KZ6RT4375JGPVNX2DJJKN7E7
kind: task
created: 2026-08-04T16:13:21Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# A detached run is never finalized, so a silent agent and a working one look identical
## So that
an agent that produced nothing is loud at the moment it exits instead of discovered turns later
## Acceptance
- [ ] a detached child's run is finalized when the process exits, not only when dacli wait is called
- [ ] outcome.md never stays at 'running' for a process that is gone
- [ ] a run that left no events and no checked acceptance is reported as 'no visible result' by agents, not only by wait
## Log
- 2026-08-05T13:59:42Z claimed by a-junior-mhqc64
