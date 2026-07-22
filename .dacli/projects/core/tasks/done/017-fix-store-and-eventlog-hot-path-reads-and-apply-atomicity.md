---
id: t-01KY4R55V4RFJZW0D1GQT2B5JX
kind: task
created: 2026-07-22T11:07:44Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# Fix store and eventlog hot-path reads and apply-atomicity

## Context
These are real self-hosted audit findings (in .dacli/projects/core/notes/findings/). Anchors:
- `store.FindTask` (store.go:357) calls `ListTasks(w,"","")` which ReadFile+Parses EVERY task; called per-pending-event in eventlog/sync.go:37, per-hit in store/taint.go canonRef:116. Build an id|seq|slug→task index once and reuse it.
- Single-item `LoadRole` (roles.go:113), `LoadRuntime` (runtimefiles.go:132), `LoadShortcut` (shortcutfiles.go:89) delegate to LoadAll* and scan the whole dir though the exact path is computable (w.RolePath(name) …). Read the single named file.
- `ListNotes` (store.go:493), `ListRisks` (risk.go:69), `LoadRoles` (roles.go:74), `LoadRuntimes` (runtimefiles.go:98), `LoadShortcuts` (shortcutfiles.go:53), `ListQueues` (queue.go:104) return nil,nil on ANY ReadDir error, hiding real I/O errors as "empty". `ListProjects` (store.go:125) does it right (os.IsNotExist check) — follow it.
- `eventlog.apply` (sync.go:67-115) commits side effects before MarkApplied (sync.go:58); a mid-apply failure re-runs from the top and duplicates a claim Log line / finding note. Make apply idempotent (check-before-write) or mark-applied atomically.
- `Taint` loop omits `model.NoteMetric`; its `strings.Contains` match is case-sensitive with no path normalization (taint.go).

## Scope (STRICT) — touch ONLY:
- `internal/store/**`
- `internal/eventlog/**`

## Staging discipline (IMPORTANT)
The working tree holds UNRELATED uncommitted state (other findings, other tasks). Do NOT `git add -A`. `git add` ONLY files under the two scope dirs above plus this task's own file under `.dacli/projects/core/tasks/`. Commit via `dacli commit` (author attribution). Run `go build ./...` and `go test ./internal/...` (green) before committing; paste the summary as `dacli note add finding`. Then `dacli task check` the boxes you satisfied.

## Acceptance
- [x] FindTask reuses a built index instead of re-reading the whole task tree per call (store.go ListTasks); kills the O(events x tasks) blowup in sync/taint loops
- [x] single-item LoadRole/LoadRuntime/LoadShortcut read their exact file, not the whole directory
- [x] ListNotes/ListRisks/LoadRoles/LoadRuntimes/LoadShortcuts/ListQueues distinguish dir-absent from a real ReadDir error (follow ListProjects)
- [x] eventlog.apply is idempotent or marks-applied atomically so a mid-apply failure cannot duplicate notes/log lines on re-sync
- [x] taint includes NoteMetric and is case/path-insensitive
- [x] committed on branch by an agent; go build + go test green
## Log
- 2026-07-22T11:46:22Z completed by a-root
