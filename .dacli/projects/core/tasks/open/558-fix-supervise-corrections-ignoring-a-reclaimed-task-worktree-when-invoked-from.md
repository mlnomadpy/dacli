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
- [ ] internal/features/execution/spawn_worktree_test.go reproduces cmdSupervise invoked with ctx.Cwd equal to w.Root after a terminal run transfers the task's registered worktree to root
- [ ] The correction runtime executes in the task's registered canonical-branch worktree and each correction run records that checkout in worktree.txt
- [ ] An unrelated registered worktree does not redirect a main-checkout supervise invocation, and ambiguous or branch-mismatched candidates still refuse with exit 3
- [ ] GOCACHE=/tmp/dacli-audit-cache go test ./internal/features/execution -run 'TestSuperviseCorrection|TestResolveSpawnWorkDir' -count=1 passes
## Log
- 2026-08-28T14:00:03Z takeover by a-root from a-adversarial-reviewer-kr8wnz (recovery: task takeover --force; reason: reviewer run completed; root reviewed the evidence and is organizing the follow-up)
