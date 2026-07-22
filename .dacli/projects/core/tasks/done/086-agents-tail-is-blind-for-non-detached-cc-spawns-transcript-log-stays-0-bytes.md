---
id: t-01KY63ECPTDQS5YGT5EHVB27RP
kind: task
created: 2026-07-22T23:44:15Z
created_by: a-root
owner: a-root
priority: should
---
# agents --tail is blind for non-detached cc spawns: transcript.log stays 0 bytes
## Acceptance
- [x] Non-detached spawn tees the child transcript to runs/<id>/transcript.log so 'agents --tail' shows the last line
- [x] A hung vs thinking distinction is observable during a synchronous reviewPhase spawn
## Log
- 2026-07-22T23:48:18Z completed by a-root
