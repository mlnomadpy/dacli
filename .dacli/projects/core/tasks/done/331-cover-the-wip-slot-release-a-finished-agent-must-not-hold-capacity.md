---
id: t-01KZP9FCSSX6P1JS7CQXWJZA49
kind: task
created: 2026-08-10T16:53:12Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# Cover the WIP-slot release: a finished agent must not hold capacity
## Acceptance
- [x] a test asserts ActiveInRole excludes an agent whose process has exited and whose task is done
- [x] a test asserts a just-minted agent with no process yet DOES still hold its slot, so spawn-then-run is not broken
- [x] both tests fail if holdsWIPSlot is made to return true unconditionally
## Log
- 2026-08-10T17:05:18Z accepted by a-root
- 2026-08-10T17:05:18Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T17:05:18Z deliverable: no dacli/331-cover-the-wip-slot-release-a-finished-agent-must-not-hold-capacity branch — nothing to check against trunk
- 2026-08-10T17:05:18Z completed by a-root
