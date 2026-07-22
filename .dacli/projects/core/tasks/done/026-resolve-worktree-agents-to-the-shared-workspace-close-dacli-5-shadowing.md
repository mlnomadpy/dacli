---
id: t-01KY4ZJQHVD81PB7KKPW8Z2JF4
kind: task
created: 2026-07-22T13:17:29Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 3, pessimistic: 5}
---
# Resolve worktree agents to the shared workspace (close dacli#5 shadowing)
## Context
The feature code is ALREADY WRITTEN and passing tests in the working tree — verify and commit it, do NOT re-implement. It closes the worktree/.dacli shadowing class (issue #1 / #5): `workspace.Find` now redirects a linked-worktree cwd to the main worktree's `.dacli`, and `spawn --worktree` no longer copies the agent file in.

## Staging discipline
Do NOT `git add -A`. `git add` ONLY these files, then `dacli commit --no-add`:
- internal/workspace/workspace.go
- internal/workspace/workspace_test.go
- internal/features/execution/execution.go
(plus this task's own file under .dacli/projects/core/tasks/)

## Steps
1. `go build ./...` then `go test ./internal/...` — both green (incl. the new internal/workspace test). Paste the summary as `dacli note add finding`.
2. `git add` ONLY the files above, then `dacli commit --no-add "<message>"`.
3. `dacli task check` each satisfied box.

## Acceptance
- [x] workspace.Find redirects a linked-worktree cwd to the MAIN worktree .dacli via git-common-dir; new workspace_test proves it
- [x] spawn --worktree no longer needs to copy the agent file in; a worktree child self-commits AND self-checks boxes against the shared workspace
- [x] committed on branch by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T13:20:08Z completed by a-root
