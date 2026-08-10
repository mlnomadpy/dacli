---
id: t-01KZP43J5B38H0KEEBHXDRZ18P
kind: task
created: 2026-08-10T15:19:22Z
created_by: a-go-auditor-qz3zb9
owner: a-go-auditor-qz3zb9
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 4}"
---
# an unknown --status value lists zero tasks and exits 0 instead of refusing, so a typo reads as an empty backlog
## Context
store.listTasksRaw (internal/store/store.go:833-836) filters model.AllStatuses with 'if status != "" && st != status { continue }'. An unrecognized status value matches none of the four canonical statuses (model.go:100-107, which has no Valid() method), so every folder is skipped and the call returns an empty slice with nil error. The command layer passes raw user input unvalidated: task list (planning.go:234), task list --json (planning.go:277), lint (insight.go:106) all do model.Status(f.Get("status")). Repro: 'dacli task list --status closed' (user means done) prints nothing, exit 0 — reads as 'backlog empty'; 'dacli lint --status opne' reports a clean sweep having examined nothing. f.Reject already refuses unknown flag NAMES loudly; an unknown flag VALUE deserves the same, not a plausible-but-wrong empty answer. See finding 'unknown --status silently lists zero tasks'.
## Acceptance
- [ ] an unrecognized --status value is refused (usage/exit 2 or a refusal) naming the bad value and the allowed set {open,active,blocked,done}, rather than returning an empty exit-0 list
- [ ] the guard covers task list, task list --json, and lint --status; an empty --status (meaning all) and each valid status still work exactly as before
- [ ] a test reproduces the bug: listing/linting with an invalid status returns a non-nil error (or non-zero exit), and asserts a valid status and empty status still succeed
## Log
