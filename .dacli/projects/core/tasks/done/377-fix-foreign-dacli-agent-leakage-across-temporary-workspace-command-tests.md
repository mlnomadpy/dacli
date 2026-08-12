---
id: t-01KZV5ERT9JTA7Y4QQ7KVQ04V3
kind: task
created: 2026-08-12T14:19:10Z
created_by: a-codex-loop-auditor-g1st46
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
github:
  issue: 469
  repo: mlnomadpy/dacli
---
# Fix foreign DACLI_AGENT leakage across temporary-workspace command tests
## Acceptance
- [x] With DACLI_AGENT set to a valid token from a different workspace, go test ./internal/features/briefing ./internal/features/orchestration ./internal/features/teamops exits 0 instead of failing with 'agent token not recognized in this workspace'
- [x] Each affected package establishes an explicit root-test identity baseline by unsetting and restoring DACLI_AGENT in its test harness; individual tests do not depend on the shell that launched go test
- [x] A regression test or package-level harness check proves a non-empty foreign ambient token cannot reach an in-process command against a fresh temporary workspace
## Log
- 2026-08-12T14:23:29Z adopted by a-root (owner a-codex-loop-auditor-g1st46 orphaned)
- 2026-08-12T14:27:50Z claimed by a-root
- 2026-08-12T15:12:08Z completed by a-root
