---
id: t-01KY53QHG2G4WGQ4XNF3XT8D23
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# E2: claim-scoped commits — dacli commit refuses files outside the task claim
## Context
E2 makes `--claim` load-bearing at commit time, killing the "do NOT git add -A" boilerplate every brief needs today (agents still slipped — 020/025 staged their agent file).

Anchors:
- `internal/features/vcs/vcs.go` `cmdCommit` (:65). After staging (or before, if not --no-add), read the committing agent's claim and REFUSE when staged files fall outside it.
- Finding the claim: the spawn wrote it to the run record. `internal/procmon` (a shared entity vcs MAY import) has `Record.Claims` and `ReadRecord`. Scan `w.RunsDir()` for the proc.txt whose `child` == the committing agent's id (`id.ID`) with a non-empty `Claims`; that is this agent's declared scope. `procmon.PathsOverlap` (or a simple "is path under any claim" check) tells you whether a staged path is in scope.
- Allowed beyond the claim: the task's own file under `.dacli/projects/.../tasks/` and (optionally) other `.dacli/` record files — do not fight the workspace record. The enforcement targets CODE files outside the claim.
- `--force` overrides with a loud stderr note naming the out-of-scope files. If the agent has no recorded claim, warn once and proceed (do not hard-block unclaimed agents).

## Scope (STRICT) — touch ONLY:
- `internal/features/vcs/vcs.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY vcs.go plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` summary, then `dacli commit`. (Yes — dogfood: your own commit should satisfy the very check you are adding.) Box-checking is owner-only; file a completion finding.

## Acceptance
- [x] dacli commit warns or refuses when staged files fall outside the spawn's declared --claim scope (plus the task file), killing the git-add-A staging-boilerplate class
- [x] the spawn's claim is recorded where commit can read it; --force overrides with a loud note
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T14:52:58Z accepted by a-root
