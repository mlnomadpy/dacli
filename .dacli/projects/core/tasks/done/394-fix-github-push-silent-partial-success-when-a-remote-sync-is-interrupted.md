---
id: t-01KZVJWQCW3QMC17P03W9YCEHF
kind: task
created: 2026-08-12T18:13:58Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 506
  repo: mlnomadpy/dacli
---
# Fix GitHub push silent partial success when a remote sync is interrupted
## So that
operators can trust a zero exit and summary to mean every planned GitHub mutation completed
## Acceptance
- [x] An injected interruption after finding comments makes github push exit nonzero and identify the incomplete task, closure, and decision stages
- [x] A successful non-dry-run always prints a final applied summary; it never exits zero with only the plan line
- [x] Re-running after an interrupted partial apply uses markers idempotently and completes all remaining mutations without duplicate issues, comments, or decisions
- [x] An integration test reproduces the observed long-window partial apply and proves both the failure signal and recovery behavior
## Log
- 2026-08-12T18:24:00Z claimed by a-codex-maintainer-1weed1
- 2026-08-12T18:56:43Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/511 (event 01KZVKY5HKNAHCYM4WJ7SHWEXT)
- 2026-08-12T19:00:46Z accepted by a-root
- 2026-08-12T19:00:46Z verified by `GOCACHE=/private/tmp/dacli-gocache GOTMPDIR=/private/tmp go test ./internal/features/ghmirror` (exit 0) in branch main at 6ea0f9e — proves that tree builds, not that the work is in trunk
- 2026-08-12T19:00:46Z deliverable: dacli/394-fix-github-push-silent-partial-success-when-a-remote-sync-is-interrupted exists but is NOT in main — closed anyway
- 2026-08-12T19:00:46Z completed by a-root
- 2026-08-12T19:44:08Z accepted by a-root
- 2026-08-12T19:44:08Z closed WITHOUT verification — no --verify command was given
- 2026-08-12T19:44:08Z deliverable: dacli/394-fix-github-push-silent-partial-success-when-a-remote-sync-is-interrupted is merged into main
- 2026-08-12T19:44:08Z completed by a-root
