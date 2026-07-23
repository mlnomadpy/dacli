---
id: t-01KY757X5FTPRG44CYKSTSWBQT
kind: task
created: 2026-07-23T09:34:54Z
created_by: a-root
owner: a-root
priority: should
---
# Add orchestration integration test for the idle->review->build transition
## Acceptance
- [ ] A driver test with a fake runner drives an empty backlog through idle+review (task filed) then a build cycle
- [ ] Test is deterministic (no real spawns)
## Log
