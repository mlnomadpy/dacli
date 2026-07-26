---
id: t-01KYG50828G43VR35Y99TKQ84K
kind: task
created: 2026-07-26T21:23:53Z
created_by: a-root
owner: a-root
priority: should
---
# ci.yml: dedup the two frontend node setups (135 build + 154 test/lint) into one job step
## So that
CI does not run npm ci + vite build twice per run
## Acceptance
- [x] ci.yml has a single node setup + one npm ci/build, with test:unit and lint added as steps; ci stays green
## Log
- 2026-07-26T21:30:46Z claimed by a-a6t1k0z52j
- 2026-07-26T21:33:04Z accepted by a-root
- 2026-07-26T21:33:04Z completed by a-root
