---
id: t-01KYHWEB1N49WH20AZX606S2RR
kind: task
created: 2026-07-27T13:32:47Z
created_by: a-root
owner: a-root
priority: must
---
# Gate privileged dacli subcommands on the agent grant
## So that
a read-only child agent cannot escalate to operator shell or writes
## Acceptance
- [ ] shortcut add, runtime add, project add/rm, pr status, kill, report, escalate require rw
- [ ] a ro agent is refused with exit 3 on each
## Log
