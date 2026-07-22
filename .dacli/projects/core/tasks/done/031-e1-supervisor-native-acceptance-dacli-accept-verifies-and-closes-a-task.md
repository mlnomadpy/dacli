---
id: t-01KY53QHFJ381DVNKHSHPFFJ56
kind: task
created: 2026-07-22T14:30:00Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# E1: supervisor-native acceptance — dacli accept verifies and closes a task
## Context
E1 removes the biggest operator friction: today the owner hand-closes EVERY task (`task check --all` + `task done`) after an agent finishes. `dacli accept <task>` does it in one verified step.

Build it as a NEW SLICE so it stays off the shared execution.go (parallel-safety): `internal/features/acceptance/acceptance.go` with a `Commands` table, registered in `internal/cli/cli.go`'s `aggregate(...)` list (add `acceptance.Commands` + the import — cli.go is the ONLY app-layer file you touch).

Behaviour of `dacli accept <task> [--verify "<cmd>"] [--all]`:
- Optionally run a verification command (e.g. `--verify "go test ./..."`); a non-zero exit REFUSES the accept (exit 1) and reports it — never close a task whose checks fail.
- Apply the acceptance boxes and move the task to done, reusing store primitives: `t.Acceptance()` (store.go:173) returns the checkboxes, set them checked and `store.SaveTask` (store.go:471), then `store.MoveTask(w, t, model.StatusDone)` (store.go:475). Mirror how `planning.go:256 cmdTaskCheck` does it — but you CANNOT import the planning slice; replicate via store only, adding a small `store.CheckAllAcceptance(t)` helper in store.go if needed (you own store.go this wave).
- Owner-only, like task check/done.
- Also support the PROPOSE path: an agent emits box-check intentions as events (a new event kind or a finding convention) that `dacli accept`/`sync` applies — so the child proposes, the owner accepts. Keep this minimal but real.

## Scope (STRICT) — touch ONLY:
- `internal/features/acceptance/` (new slice)
- `internal/store/store.go` (only if you need a CheckAllAcceptance helper)
- `internal/cli/cli.go` (register the slice)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the files above plus this task's file. `go build ./...` + `go test ./internal/...` green (the arch_test enforces slice isolation — do not import another feature slice). `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only; file a completion finding, do not retry a refused check.

## Acceptance
- [x] dacli accept <task> verifies the agent's completion (build/test hook + acceptance criteria) and applies box-checks + done in one step, so the owner sets policy instead of hand-closing every spawn
- [x] an agent can PROPOSE box-checks as events that dacli sync/accept applies (owner still owns the decision), removing the per-task manual close
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T14:52:58Z accepted by a-root
