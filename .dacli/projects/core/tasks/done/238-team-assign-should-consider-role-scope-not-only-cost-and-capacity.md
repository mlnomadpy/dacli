---
id: t-01KZ52VGBET1FR3MVPFJB94YMA
kind: task
created: 2026-08-04T00:30:24Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# team assign should consider role scope not only cost and capacity
## So that
a task does not route to a domain-inappropriate role on an alphabetical tie
## Acceptance
- [x] scope overlap with the task's files breaks ties before name does
## Log
- 2026-08-04T10:49:53Z claimed by a-maintainer-wvxmkf
- 2026-08-04T11:33:01Z accepted by a-root
- 2026-08-04T11:33:01Z verified by `go test ./internal/team/ ./internal/features/teamops/` (exit 0)
- 2026-08-04T11:33:01Z completed by a-root
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/296 (event 01KZ67C42XTT6P9XT042YCPGQF)
