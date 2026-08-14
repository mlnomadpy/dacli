---
id: t-01KZYBZZHRQ77PZQSPCCYTTD66
kind: task
created: 2026-08-13T20:11:08Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 614
  repo: mlnomadpy/dacli
---
# Fix wait finalization after agents recovers a detached process
## So that
a status read cannot make dacli wait hang on an already-dead worker
## Acceptance
- [x] internal/features/execution uses one terminal lifecycle state so agents recovery and wait agree after proc.txt gains outcome: process exited (recovered)
- [x] a regression test calls agents before wait for a dead detached run and wait finalizes outcome.md without timing out
- [x] the test also covers multiple named recovered runs and releases every recorded path claim
## Log
- 2026-08-13T21:02:14Z claimed by a-codex-maintainer-8r5s5s
- 2026-08-13T21:16:13Z accepted by a-root
- 2026-08-13T21:16:13Z verified by `go test ./internal/features/execution -run TestAgentsRecoveryLetsWaitFinalizeMultipleNamedRuns -count=1` (exit 0) in branch main at ca2bb20 — proves that tree builds, not that the work is in trunk
- 2026-08-13T21:16:13Z deliverable: dacli/436-fix-wait-finalization-after-agents-recovers-a-detached-process is merged into main
- 2026-08-13T21:16:13Z completed by a-root
- 2026-08-13T23:51:31Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/624 (event 01KZYFBEGHSRVS5B89Y0ABPYSD)
