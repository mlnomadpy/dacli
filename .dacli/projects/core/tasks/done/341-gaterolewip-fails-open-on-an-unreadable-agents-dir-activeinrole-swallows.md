---
id: t-01KZPJE3X9TXRVTDYMCAGX54CC
kind: task
created: 2026-08-10T19:29:47Z
created_by: a-go-auditor-vek0m1
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# gateRoleWIP fails open on an unreadable agents dir: ActiveInRole swallows ListAgents' error and returns 0
## Acceptance
- [x] ActiveInRole (internal/store/roles.go:210) no longer swallows the ListAgents error: it either returns (int, error) or otherwise surfaces an unreadable-agents-dir failure to its caller instead of silently returning 0
- [x] gateRoleWIP (internal/features/execution/execution.go:521) fails CLOSED when the agents dir cannot be read — it refuses the spawn rather than passing, matching the sibling gateClaimOverlap precedent (execution.go:612-620, task 337)
- [x] A red-green test drives gateRoleWIP with the agents dir replaced by a regular file (ENOTDIR, the technique used in 337's TestGateClaimOverlapFailsClosedOnUnreadableRunsDir) and asserts a non-zero exit / refusal, not a pass
- [x] All other ActiveInRole callers (insight.go:1153, teamops.go:62/547, dashboard/roster.go) still compile and behave unchanged on a readable agents dir; go build ./... and go vet ./... clean
## Log
- 2026-08-10T19:51:39Z adopted by a-root (owner a-go-auditor-vek0m1 orphaned)
- 2026-08-10T19:51:39Z claimed by a-fixer-2mp87m
- 2026-08-10T20:02:03Z accepted by a-root
- 2026-08-10T20:02:03Z verified by `go test ./internal/features/teamops/ ./internal/features/execution/ ./internal/store/` (exit 0)
- 2026-08-10T20:02:03Z deliverable: dacli/341-gaterolewip-fails-open-on-an-unreadable-agents-dir-activeinrole-swallows exists but is NOT in sprint/4 — closed anyway
- 2026-08-10T20:02:03Z completed by a-root
