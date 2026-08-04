---
id: t-01KZ6RTFZEG3HQDB2ZEXV7MBE4
kind: task
created: 2026-08-04T16:13:34Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# An agent blocked from running dacli has no way to say so, because every escape hatch is dacli
## So that
the one failure that silences every other report is still reportable
## Acceptance
- [ ] a child can record 'I am blocked and why' using only a plain file write, with no dacli invocation
- [ ] agents and wait surface that report as a distinct state, not as a normal completion
- [ ] the child prompt tells the agent this channel exists and when to use it
## Log
