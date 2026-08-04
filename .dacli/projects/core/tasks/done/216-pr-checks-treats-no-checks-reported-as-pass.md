---
id: t-01KZ4WCR6MKFDPFFAW0C2GPHG5
kind: task
created: 2026-08-03T22:37:29Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# pr checks treats no checks reported as pass
## So that
a repo with no CI cannot merge everything green
## Acceptance
- [x] absent checks are distinguishable from passing checks
## Log
- 2026-08-03T22:49:24Z claimed by a-62feb0eqzq
- 2026-08-04T00:27:45Z accepted by a-root
- 2026-08-04T00:27:45Z verified by `go build ./...` (exit 0)
- 2026-08-04T00:27:45Z completed by a-root
- 2026-08-04T00:37:40Z a-fixer-2675tw: PR opened: https://github.com/mlnomadpy/dacli/pull/278 (event 01KZ51QC83J8QR0AYSVTHBM8DN)
