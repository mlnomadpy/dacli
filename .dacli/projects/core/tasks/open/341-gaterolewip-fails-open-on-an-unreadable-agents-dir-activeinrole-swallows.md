---
id: t-01KZPJE3X9TXRVTDYMCAGX54CC
kind: task
created: 2026-08-10T19:29:47Z
created_by: a-go-auditor-vek0m1
owner: a-go-auditor-vek0m1
priority: should
---
# gateRoleWIP fails open on an unreadable agents dir: ActiveInRole swallows ListAgents' error and returns 0
## Acceptance
- [ ] ActiveInRole (internal/store/roles.go:210) no longer swallows the ListAgents error: it either returns (int, error) or otherwise surfaces an unreadable-agents-dir failure to its caller instead of silently returning 0
- [ ] gateRoleWIP (internal/features/execution/execution.go:521) fails CLOSED when the agents dir cannot be read — it refuses the spawn rather than passing, matching the sibling gateClaimOverlap precedent (execution.go:612-620, task 337)
- [ ] A red-green test drives gateRoleWIP with the agents dir replaced by a regular file (ENOTDIR, the technique used in 337's TestGateClaimOverlapFailsClosedOnUnreadableRunsDir) and asserts a non-zero exit / refusal, not a pass
- [ ] All other ActiveInRole callers (insight.go:1153, teamops.go:62/547, dashboard/roster.go) still compile and behave unchanged on a readable agents dir; go build ./... and go vet ./... clean
## Log
