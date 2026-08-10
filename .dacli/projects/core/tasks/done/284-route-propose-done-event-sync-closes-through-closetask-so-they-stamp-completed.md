---
id: t-01KZ6SPYCAJG744JHC3DSYK7JJ
kind: task
created: 2026-08-04T16:29:06Z
created_by: a-go-auditor-s12cpg
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Route propose:done event-sync closes through CloseTask so they stamp completed-by and verify acceptance
## So that
a task closed by a non-owner's proposal is stamped and verified exactly like the owner's direct close, so calibration and doctor see it and no unmet-acceptance task lands in done/
## Context
Found by task 279 audit. A non-owner 'dacli task done <ref>' files EventProposeStatus 'propose: done' (planning.go:389), returning before the owner-path acceptance check. eventlog.Sync auto-applies it between supervisor turns (execution.go:1017); apply() calls store.MoveTask(w,t,done) directly (sync.go:146-164), bypassing store.CloseTask (store.go:1188-1201) — the canonical close from task 037 that stamps 'completed by' so 'no path can close a task without the stamp calibration reads'. Result: done task with no actuals stamp (excluded from calibration.go:155/368 samples, flagged by doctor) and, if acceptance is unmet, a 'done' no check supports. Contrast the sibling accept-propose path (sync.go:166-176), which Sync leaves pending for verified 'dacli accept'. Scope: internal/eventlog/sync.go (+ a shared close helper if CloseTask must be reachable without an import cycle) and its test. Do NOT broaden into a rewrite of the proposal model.
## Acceptance
- [x] eventlog.Sync apply() for EventProposeStatus targeting model.StatusDone routes through store.CloseTask (writing a 'completed by' stamp) instead of a bare store.MoveTask (sync.go:146-164)
- [x] a task closed via a propose:done event has its acceptance boxes verified before the move: unmet acceptance is NOT silently moved to done — it mirrors the owner path's refusal (planning.go:396-406), leaving the event pending or recording the refusal
- [x] store.calibration.logSpan can measure a task closed via the proposal path (a 'completed by' stamp exists) and store.LogHasStamp(t,'completed by') returns true; doctor no longer reports it as a broken claim->completion span
- [x] regression test: a non-owner 'task done' on a task with all boxes checked, followed by owner eventlog.Sync, yields the same Log stamps + calibration span as the owner's direct CloseTask; and the same with an UNCHECKED box does NOT leave the task in done/
- [x] go build ./... and go test ./internal/... green
## Log
- 2026-08-05T13:02:38Z adopted by a-root (owner a-go-auditor-s12cpg orphaned)
- 2026-08-05T13:02:38Z accepted by a-root
- 2026-08-05T13:02:38Z closed WITHOUT verification — no --verify command was given
- 2026-08-05T13:02:38Z completed by a-root
- 2026-08-08T11:07:20Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/364 (event 01KZ77GJT87G76C7WQ8T6PE3ZM)
