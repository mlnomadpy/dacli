---
id: t-01KY757X3WZKWBB6NS89QAY8A6
kind: task
created: 2026-07-23T09:34:54Z
created_by: a-root
owner: a-root
priority: should
github:
  issue: 197
  repo: mlnomadpy/dacli
---
# Fix misleading '(read-only grant)' messages: the real reason is non-ownership, not grant
## Acceptance
- [x] The 4 task-mutation messages distinguish 'not the owner (propose an event, or accept --force as root)' from an actual ro-grant refusal
- [x] Wording verified; no behavior change beyond the message
## Log
- 2026-07-23T10:49:44Z claimed by a-n9n6r0nn4w
- 2026-07-23T10:52:30Z accepted by a-root
- 2026-07-23T10:52:30Z completed by a-root
- 2026-08-03T22:38:15Z a-n9n6r0nn4w: PR opened: https://github.com/mlnomadpy/dacli/pull/57 (event 01KY79NJ3PQ8T81MKD1R5WD900)
