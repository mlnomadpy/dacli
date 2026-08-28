---
id: t-01M13V42QDZE7CKYDFWYVB5YG5
kind: task
created: 2026-08-28T09:27:25Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 848
  repo: mlnomadpy/dacli
---
# Fix supervise correction agents lacking commit authority in root-reclaimed worktrees
## So that
bounded correction loops can finish verified work instead of preserving an uncommittable staged patch
## Acceptance
- [x] A fixture creates a terminal child worktree, reclaims exact paths to root, and runs supervise on the same task without an inevitable commit-policy refusal.
- [x] The supervised correction is either granted an audited task-scoped transfer it can commit under or materialized by root with preserved child attribution and verification evidence.
- [x] Unrelated agents and paths outside the exact claim remain refused.
- [x] A two-turn regression proves supervise does not spend every turn repeating the same ownership refusal.
- [x] Mutation evidence and focused orchestration/VCS tests pass.
## Log
- 2026-08-28T09:28:52Z claimed by a-fixer-5xjt19
- 2026-08-28T09:50:17Z accepted by a-root
- 2026-08-28T09:50:17Z verified by `env GOCACHE=/tmp/dacli-root-532-accept-cache go test ./internal/cli ./internal/features/execution -run TestSuperviseCorrection -count=1` (exit 0) in branch dacli/532-fix-supervise-correction-agents-lacking-commit-authority-in-root-reclaimed at 2e3d6968 — proves that tree builds, not that the work is in trunk
- 2026-08-28T09:50:17Z deliverable: dacli/532-fix-supervise-correction-agents-lacking-commit-authority-in-root-reclaimed exists but is NOT in main — closed anyway
- 2026-08-28T09:50:17Z completed by a-root
- 2026-08-28T09:51:42Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/849 (event 01M13VMAHGQ3B3R0EGZ14W0YGE)
- 2026-08-28T09:51:42Z a-root: Landing policy override: mode=pr base=main (event 01M13WEJWG6YW31ET1W6PF0JVC)
- 2026-08-28T09:51:42Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/849 at merge commit 27f25268f85af647230c0d4de4aef342ea041dd3 into main (generation 0) (event 01M13WETYKWEHNH796FBF2T0ZS)
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-root-532-accept-cache go test ./internal/cli ./internal/features/execution -run TestSuperviseCorrection -count=1","exit_code":0,"duration_ms":8129,"artifact_hash":"sha256:22c22a778f696d72552068b8f0e955e712ed154982fa3e7b6fe9e24324cf7d72","verifier":"a-root","branch":"dacli/532-fix-supervise-correction-agents-lacking-commit-authority-in-root-reclaimed","commit_sha":"2e3d69682bb93370d3cf7afab8388133205d6fc8"}
