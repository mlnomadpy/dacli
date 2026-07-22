---
id: t-01KY59FNFK27A1084PQ8R2CJ5S
kind: task
created: 2026-07-22T16:10:34Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT R6: remaining feature slices
## Acceptance
- [x] findings filed with file:line in internal/features/{collab,teamops,knowledge,onboard,queues,shortcuts,skillforge,ghmirror,governance,wscore,selfreport} and internal/skills
## Log
- 2026-07-22T16:11:07Z claimed by a-9y38s7w8e2
- 2026-07-22T16:17:27Z finding by a-9y38s7w8e2: dacli init silently ignores --template and --roster: two spec-documented, Brief-advertised flags do nothing (event 01KY59QDNF14BCXYP41Y70XNJ4)
- 2026-07-22T16:17:27Z finding by a-9y38s7w8e2: governance slice docstring is stale: claims to stub subsystems that are all now built (event 01KY59QM9M7663NKC9X5WSF72B)
- 2026-07-22T16:17:27Z finding by a-9y38s7w8e2: ghmirror marker-idempotency can still duplicate: recovery leans on eventually-consistent GitHub search (event 01KY59QXYZRND966AN1CWK7NM5)
- 2026-07-22T16:17:27Z finding by a-9y38s7w8e2: collab threads mis-attributes answers: answered-by is keyed per-task, not per-question (event 01KY59R55HBYATM4VGPT59K092)
- 2026-07-22T16:17:27Z finding by a-9y38s7w8e2: selfreport gh subprocesses have no context timeout — a hung gh blocks dacli (incl. mcp serve) (event 01KY59RD05V5AY73TPTDRP41WC)
- 2026-07-22T16:17:27Z finding by a-9y38s7w8e2: AUDIT R6 coverage: all 4 planned() stubs are HONEST; clean slices vs. the 5 gaps filed (event 01KY59RW6XY2325A7BMRQ435HH)
- 2026-07-22T18:52:27Z accepted by a-root
- 2026-07-22T18:52:27Z completed by a-root
