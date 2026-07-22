---
id: t-01KY4KWP69XPM5CMTR1HFP9QXM
kind: task
created: 2026-07-22T09:53:12Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# Resource monitoring and kill for spawned agent process trees

## Context
The feature code is ALREADY WRITTEN and passing tests in the working tree — your job is to verify and commit it, NOT to re-implement it. The change:
- new `internal/procmon/procmon.go` (+ `procmon_test.go`): process-group liveness, `ps`-based RAM/CPU sampling, best-effort nvidia GPU, `KillTree` (SIGTERM→SIGKILL).
- `internal/features/execution/execution.go`: `execRuntime` now sets `Setpgid`, kills the whole group on timeout, and registers a `proc.txt`; new `dacli agents` and `dacli kill` commands.
- `internal/features/execution/verify.go`: verify-panel spawns register their proc record too.

## Staging discipline (IMPORTANT)
The working tree also holds UNRELATED uncommitted state from an earlier parallel run (many `.dacli/projects/core/notes/findings/*`, new agent files, other roles). DO NOT `git add -A`. Stage ONLY these paths:
- `internal/procmon/`
- `internal/features/execution/execution.go`
- `internal/features/execution/verify.go`
- `.dacli/roles/maintainer.md`
- `.dacli/projects/core/tasks/` (this task's file, wherever it now lives)

## Steps
1. `go build ./...` and `go test ./internal/...` — confirm green; paste the summary as a `dacli note add finding`.
2. Create the branch, `git add` ONLY the paths above, and commit via `dacli commit` (so authorship is attributed to you, the maintainer role).
3. Check each acceptance box you have satisfied with `dacli task check`.

## Acceptance
- [x] procmon gives every spawn a killable process group (Setpgid); timeout SIGKILLs the whole group not just the leader
- [x] dacli agents lists only live agents with their tree's RAM/CPU/GPU, procs, and uptime; GPU honestly n/a when unmeasurable
- [x] dacli kill <ref|--all> reaps the whole tree (SIGTERM then SIGKILL after grace), leaves no runaway children, writes an audit crumb
- [x] committed on a branch by the maintainer agent with go build + go test green
## Log
- 2026-07-22T10:49:32Z completed by a-root
