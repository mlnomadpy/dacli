---
id: t-01KY4VAQ4QTF2DFSXGCQNS6SAK
kind: task
created: 2026-07-22T12:03:12Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# Async spawn/wait + agent-lifecycle features from my dacli feedback

## Context
The feature code is ALREADY WRITTEN and passing tests in the working tree — verify and commit it, do NOT re-implement. It implements 8 items of operator feedback (async orchestration, worktree attribution fix, headless contract, salience/slug/attribution hygiene, logs -f, path-claim, budget-reap).

## Staging discipline (IMPORTANT)
The working tree also holds unrelated `.dacli/**` workspace churn. Do NOT `git add -A`. `git add` ONLY these code files, then `dacli commit --no-add`:
- internal/features/execution/execution.go
- internal/features/execution/verify.go
- internal/features/vcs/vcs.go
- internal/procmon/procmon.go
- internal/prompts/tpl/protocol_preamble.md
- internal/store/store.go
- internal/spm/criticalpath.go
- internal/cli/main_test.go
(plus this task's own file under .dacli/projects/core/tasks/)

## Steps
1. `go build ./...` then `go test ./internal/...` — both must be green (the new cli TestMain now clears DACLI_AGENT, so the suite passes even though you are a dacli agent). Paste the summary as `dacli note add finding`.
2. `git add` ONLY the files above, then `dacli commit --no-add "<message>"` (author attribution).
3. `dacli task check` each satisfied box.

## Acceptance
- [ ] spawn --detach returns a run-id immediately; dacli wait <ids|all> blocks until done and finalizes outcome from workspace effects
- [ ] worktree spawn copies the child agent-file into the worktree so it self-recognizes; outcome reads worktree events (fixes issue #1 attribution)
- [ ] headless preamble forbids waiting for approval; cli TestMain clears DACLI_AGENT; commit warns loudly when role unresolved; Slugify capped to a legal filename
- [ ] dacli logs -f follows a transcript; spawn --claim refuses overlapping live claims; dacli agents --max-rss/--max-runtime --reap kills over-budget trees
- [ ] committed on branch by an agent; go build + go test ./internal/... green
## Log
