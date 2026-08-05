---
id: t-01KZ701KR144Y0ARQM1FBKJEPN
kind: task
created: 2026-08-04T18:19:47Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 3, pessimistic: 6}"
---
# The brief truncates findings and decisions in alphabetical filename order, so a critical one can be silently dropped
## So that
an agent's brief carries the findings that matter rather than the ones whose filenames sort early
## Acceptance
- [x] the cap selects by severity, trust and recency rather than os.ReadDir order
- [x] what was dropped is named in the brief, not reported as a bare count
## Log
- 2026-08-05T13:03:22Z accepted by a-root
- 2026-08-05T13:03:22Z verified by `go build ./...` (exit 0)
- 2026-08-05T13:03:22Z completed by a-root
