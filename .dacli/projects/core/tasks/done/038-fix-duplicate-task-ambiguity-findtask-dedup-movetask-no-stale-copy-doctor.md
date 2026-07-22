---
id: t-01KY574DGYFJFFHNAG68AB81WC
kind: task
created: 2026-07-22T15:29:28Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# Fix duplicate-task ambiguity: FindTask dedup, MoveTask no-stale-copy, doctor duplicate check
## Context
Real bug, surfaced when `dacli ship` exercised the full done-task list: task 026 existed in BOTH `tasks/open/026-*` (stale, 0 boxes) and `tasks/done/026-*` (authoritative, 3 boxes), so `store.FindTask(w, "26")` errored `ref "26" is ambiguous: 026-…, 026-…` (the same task twice), breaking integrate/ship. I removed the stale file, but the code must be hardened so this can't recur or hide.

Anchors:
- `internal/store/store.go` `ListTasks` (:321) walks every status folder; when a task's file exists in two folders it yields the task TWICE. `FindTask` (:428) then reports the duplicate as an ambiguous ref. Fix: dedup by task **id** (and, when the same id appears in two statuses, prefer the terminal/most-recently-modified one — a done copy over an open one), so a stale duplicate resolves cleanly rather than erroring.
- `MoveTask` (:497) moves a task between status folders — verify it REMOVES the source file (a leftover source copy is the root cause of a duplicate). Harden it so a move can never leave a stale source copy.
- `internal/features/insight/insight.go` `cmdDoctor` (~:762) — add a check that flags any task **id/seq** appearing in more than one status folder (a duplicate task file), naming the paths, so the drift is visible.

## Scope (STRICT) — touch ONLY:
- `internal/store/store.go`
- `internal/features/insight/insight.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the files above plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` summary, then `dacli commit` (E2's claim check is live). Box-checking is owner-only.

## Acceptance
- [x] FindTask resolves unambiguously when a stale duplicate exists (dedup by task id, prefer the authoritative/done copy) instead of erroring 'ambiguous' on the same task twice
- [x] MoveTask guarantees the source-status copy is removed after a move; ListTasks never yields the same task from two status folders
- [x] dacli doctor flags duplicate task files (same seq across status folders) so the drift is visible
- [x] committed on branch by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T15:30:44Z claimed by a-7zg8j1n976
- 2026-07-22T15:38:12Z accepted by a-root
- 2026-07-22T15:38:12Z completed by a-root
