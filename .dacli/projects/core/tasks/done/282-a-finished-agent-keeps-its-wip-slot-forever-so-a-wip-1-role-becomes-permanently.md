---
id: t-01KZ6SEH9XC4GZ3ENW1R09WBGC
kind: task
created: 2026-08-04T16:24:30Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 5}"
---
# A finished agent keeps its WIP slot forever, so a wip:1 role becomes permanently unusable
## So that
a role's WIP limit bounds concurrent work rather than lifetime work
## Acceptance
- [x] an agent with no live process and no recent activity does not hold a WIP slot
- [x] doctor names any role whose WIP is held entirely by agents that are gone
- [x] the fix distinguishes retired from finished-but-never-retired rather than treating absence of status as live
## Log
- 2026-08-08T12:07:45Z accepted by a-root
- 2026-08-08T12:07:45Z verified by `go build ./...` (exit 0)
- 2026-08-08T12:07:45Z completed by a-root
