---
id: t-01KZNYHPH9DVJJYD6P0WAVFNB5
kind: task
created: 2026-08-10T13:42:13Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Audit the system as it stands after the August hardening wave and name what still needs to change
## So that
the next decisions are made against what the code is today, not against a report written before fifteen PRs landed
## Acceptance
- [x] each layer (entities, features, app, kernel) is assessed against the code at HEAD, naming file:line evidence rather than describing intent
- [x] every claim is verified against current source before it is written, and anything already fixed is reported as already fixed
- [x] findings are ranked by what an unattended run would do wrong, with the cheapest correct fix named for each
## Log
- 2026-08-10T13:43:22Z claimed by a-go-auditor-5yxchj
- 2026-08-10T13:57:45Z finding by a-go-auditor-5yxchj: prChecksPass misreads a red/pending check as GitHub-unreachable and local-merges unverified code to trunk (event 01KZNZ9ZQ3YQY518AMJV07S4A6)
- 2026-08-10T13:57:45Z finding by a-go-auditor-5yxchj: stage gates report PASS on a swallowed ListTasks/ListRisks read error (empty set => vacuous true), advancing a stage with zero tasks examined (event 01KZNZACBYG0QJ73EK0A2WXZQ3)
- 2026-08-10T13:57:45Z finding by a-go-auditor-5yxchj: loop BUILD ordering silently drops critical-path order that dacli next shows: criticalPathSlack includes the unsized loop anchor, cmdNext excludes it (event 01KZNZAPBD761KZK6PNMT2P8JM)
- 2026-08-10T14:04:16Z accepted by a-root
- 2026-08-10T14:04:16Z verified by `go test ./internal/gates/... ./internal/features/vcs/... ./internal/features/orchestration/...` (exit 0)
- 2026-08-10T14:04:16Z deliverable: dacli/313-audit-the-system-as-it-stands-after-the-august-hardening-wave-and-name-what is merged into trunk
- 2026-08-10T14:04:16Z completed by a-root
