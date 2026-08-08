---
id: t-01KYHWEWP7PNFAF2SD7WSES265
kind: task
created: 2026-07-27T13:33:05Z
created_by: a-root
owner: a-root
priority: should
---
# Adopt Flags.Reject on all command handlers
## So that
a typo'd flag fails with exit 2 instead of silently running wrong
## Acceptance
- [x] every handler rejects unknown flags except run which forwards
## Log
- 2026-07-27T23:03:03Z completed by a-root
