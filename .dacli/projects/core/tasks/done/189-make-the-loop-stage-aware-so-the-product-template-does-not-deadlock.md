---
id: t-01KZ4E0YMHRVKH7JVEJVYD4V4F
kind: task
created: 2026-08-03T18:26:22Z
created_by: a-root
owner: a-root
priority: must
---
# Make the loop stage-aware so the product template does not deadlock
## So that
a stage-gated project can actually run the loop
## Acceptance
- [x] the driver reads gate status and picks a phase-appropriate role
- [x] the loop advances a stage when its gate opens
## Log
- 2026-08-03T19:18:29Z completed by a-root
