---
id: t-01KY53QHGRNRT2NQ7BYAYJ9BWS
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 4, pessimistic: 6}
---
# E4: dacli integrate --tasks <list> --into <branch> with cleanup
## Context
E4 fixes the integration friction: every wave the operator hand-runs `git merge --no-ff` + `git worktree remove` + `git branch -D` because `dacli integrate` only scans done-status tasks, assumes `into=main`, and takes no explicit list.

Anchors:
- `internal/features/vcs/lifecycle.go` `cmdIntegrate` (:230) — extend it (or add a sibling path) so it accepts `--tasks <seq,seq,...>` (an explicit list, resolved via `resolveTaskFlag`/`store.FindTask`) and `--into <branch>` (default main, but honor any branch — the current-branch guard should compare against the given --into, which it already does). For each task: merge its `BranchFor(t)` branch via the existing `mergeTask` helper, print a per-task `merged | conflict | skipped` line, and on a CLEAN merge remove the worktree AND delete the now-merged branch. A conflict blocks that one task (existing behaviour) and integration continues to the next (or stops — keep the existing serialized-stop semantics but report clearly which merged before the stop).
- Keep the no-arg behaviour (scan all done tasks) working for back-compat.

## Scope (STRICT) — touch ONLY:
- `internal/features/vcs/lifecycle.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY lifecycle.go plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only; file a completion finding.

## Acceptance
- [x] dacli integrate takes an explicit task list and a target branch (not just into=main), merges each done-task branch, reports per-task merged/conflict, and removes the worktree + branch on success
- [x] a conflict blocks that one task and continues the rest (or stops, documented), never half-merges
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T14:52:58Z accepted by a-root
