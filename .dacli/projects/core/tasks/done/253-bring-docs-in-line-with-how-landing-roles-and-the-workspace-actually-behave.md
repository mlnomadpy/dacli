---
id: t-01KZ67G75Q13NTX1AMSRP572QS
kind: task
created: 2026-08-04T11:10:51Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2.5, pessimistic: 5}"
---
# Bring docs/ in line with how landing, roles and the workspace actually behave
## So that
the docs stop teaching a merge path and a role model that this session proved wrong
## Acceptance
- [x] GITHUB.md documents dacli integrate as the merge path and the dacli/<seq>-<slug> branch key
- [x] TEAM.md and ROSTER.md state that a role's grant and runtime must agree and how to check
- [x] SELFHOSTING.md reflects the real merged-PR count
- [x] docs/README.md index has an accurate line for every file
## Log
- 2026-08-04T11:10:57Z claimed by a-maintainer-vgqd2d
- 2026-08-04T11:33:45Z accepted by a-root
- 2026-08-04T11:33:45Z verified by `go test ./internal/features/catalog/` (exit 0)
- 2026-08-04T11:33:45Z completed by a-root
