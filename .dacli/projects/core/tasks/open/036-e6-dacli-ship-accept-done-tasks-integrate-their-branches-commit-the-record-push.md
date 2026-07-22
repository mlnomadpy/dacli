---
id: t-01KY55NX80QGRNMQNARZXAS7GQ
kind: task
created: 2026-07-22T15:04:04Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# E6: dacli ship — accept done tasks, integrate their branches, commit the record, push, in one operator command
## Context
E6 closes the last manual friction: after agents finish, the operator still hand-runs accept → integrate → a `git add -A` record commit → push, every wave. `dacli ship` does it as one command.

Build it as a NEW SLICE `internal/features/ship/ship.go` with a `Commands` table, registered in `internal/cli/cli.go`'s `aggregate(...)` (add `ship.Commands` + the import next to `acceptance.Commands` at :73 — cli.go is the ONLY app-layer file you touch). Slices can't import each other, so ship ORCHESTRATES by shelling out to its own binary (`os.Executable()`), exactly as the prompt templates tell agents to call dacli.

`dacli ship [--into <branch>] [--push] [--dry-run]`:
1. Integrate every done task's branch: shell `dacli integrate --tasks <done-refs> --into <branch>` (or the no-arg all-done form). Report per-task result; a conflict stops and reports (never half-ships).
2. Commit the workspace record SAFELY — the whole reason this exists: stage ONLY the `.dacli/` record and NEVER worktrees/runs/build. `.dacli/.gitignore` already ignores those, so `git add .dacli` is safe, but be explicit — do NOT `git add -A` (that is the operator footgun that tracked a worktree gitlink this session). Commit with a clear message.
3. `--push`: push the current branch to origin (gitx.Push or shell git push). Without `--push`, stop after the commit and print the push command.
4. `--dry-run`: print each step it WOULD run, execute nothing.

Keep it honest: any step's non-zero exit stops the pipeline and reports which steps completed.

## Scope (STRICT) — touch ONLY:
- `internal/features/ship/` (new slice)
- `internal/cli/cli.go` (register the slice)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the files above plus this task's file. `go build ./...` + `go test ./internal/...` green (arch_test forbids feature→feature imports — shell out, don't import). `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [ ] dacli ship runs accept (verified) -> integrate --tasks -> a dacli-native workspace-record commit (never sweeping worktrees/runs/build) -> push, closing the manual wave tail the operator still does by hand
- [ ] each step is skippable/dry-runnable; a failure stops and reports, never half-ships
- [ ] committed on branch by an agent; build + test green
## Log
- 2026-07-22T15:06:45Z claimed by a-2rw3qy91zz
