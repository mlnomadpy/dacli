---
id: t-01M14ABSH137KVYGMWBY9FC927
kind: task
created: 2026-08-28T13:53:47Z
created_by: a-adversarial-reviewer-kr8wnz
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 880
  repo: mlnomadpy/dacli
---
# Fix supervise corrections ignoring a reclaimed task worktree when invoked from main
## So that
root-owned recovery corrections retain the task-scoped checkout and governed commit attribution
## Acceptance
- [x] internal/features/execution/spawn_worktree_test.go reproduces cmdSupervise invoked with ctx.Cwd equal to w.Root after a terminal run transfers the task's registered worktree to root
- [x] The correction runtime executes in the task's registered canonical-branch worktree and each correction run records that checkout in worktree.txt
- [x] An unrelated registered worktree does not redirect a main-checkout supervise invocation, and ambiguous or branch-mismatched candidates still refuse with exit 3
- [x] GOCACHE=/tmp/dacli-audit-cache go test ./internal/features/execution -run 'TestSuperviseCorrection|TestResolveSpawnWorkDir' -count=1 passes
## Log
- 2026-08-28T14:00:03Z takeover by a-root from a-adversarial-reviewer-kr8wnz (recovery: task takeover --force; reason: reviewer run completed; root reviewed the evidence and is organizing the follow-up)
- 2026-08-28T15:11:17Z claimed by a-fixer-wskwrc
- 2026-08-28T15:27:16Z accepted by a-root
- 2026-08-28T15:27:16Z verified by `env -u DACLI_AGENT GOCACHE=/tmp/dacli-558-postland-cache go test ./internal/features/execution ./internal/cli -run "TestSuperviseCorrection|TestResolveSpawnWorkDir|TestSuperviseLoopConverges|TestSuperviseStalls" -count=1` (exit 0) in branch main at 67f94046 — proves that tree builds, not that the work is in trunk
- 2026-08-28T15:27:16Z deliverable: dacli/558-fix-supervise-corrections-ignoring-a-reclaimed-task-worktree-when-invoked-from is merged into main
- 2026-08-28T15:27:16Z completed by a-root
- 2026-08-28T15:27:18Z deliverable: dacli/558-fix-supervise-corrections-ignoring-a-reclaimed-task-worktree-when-invoked-from is merged into main
## Verification Evidence
{"command":"env -u DACLI_AGENT GOCACHE=/tmp/dacli-558-final-evidence go test ./internal/features/execution -run \"TestSuperviseCorrection|TestResolveSpawnWorkDir\" -count=1","exit_code":0,"duration_ms":3679,"artifact_hash":"sha256:492e8ddc045ed205a2531cd2fdd888645757879c1febede7e804d703609d09cc","verifier":"a-root","branch":"dacli/558-fix-supervise-corrections-ignoring-a-reclaimed-task-worktree-when-invoked-from","commit_sha":"f7f43d9165ae73ee2019b494922aec305859f197"}
{"command":"env -u DACLI_AGENT GOCACHE=/tmp/dacli-558-postland-cache go test ./internal/features/execution ./internal/cli -run \"TestSuperviseCorrection|TestResolveSpawnWorkDir|TestSuperviseLoopConverges|TestSuperviseStalls\" -count=1","exit_code":0,"duration_ms":5932,"artifact_hash":"sha256:2f1a97e8d664e42c0742612c9d94021aba43b2b9ed2b9457ff88a50b2163e2ce","verifier":"a-root","branch":"main","commit_sha":"67f94046268d2e5674fbdf1f05f7a6526346be32"}
