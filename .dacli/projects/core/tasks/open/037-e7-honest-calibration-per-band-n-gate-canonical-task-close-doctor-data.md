---
id: t-01KY55NX8EVAR400DQ3VF5AR9X
kind: task
created: 2026-07-22T15:04:04Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 2, probable: 4, pessimistic: 6}
---
# E7: honest calibration — per-band n-gate, canonical task-close, doctor data-integrity check
## Context
E7 fixes three integrity gaps found while building the D/E series.

Anchors:
- **Per-band n-gate.** `internal/features/insight/insight.go` `cmdCalibrate` — the by-agent band prints at :661, and the size-band already gates the "briefs now show" claim on `len(all) < 10` (:680). But a per-band line with `n=1` currently prints a `p10–p90 ×0.03–×0.03` range as if calibrated — the exact "confidence theater" the size gate warns against. Mark any band (agent OR size) with n<10 as `provisional` and DO NOT print a percentile range for it; only bands with n>=10 show the calibrated range.
- **Canonical task-close.** Two paths close a task and drifted: `planning.go:325` (`task done`) stamps `completed by` (the actuals capture field), but `acceptance.go` stamped only `accepted by` — which silently broke calibration until caught by hand. Add ONE primitive `store.CloseTask(w, t, actor)` that stamps `completed by <actor>` + MoveTask→done, and make BOTH `cmdTaskDone` and `acceptance` call it. No path may close a task without the actuals stamp. (accept may keep its extra `accepted by` line, but the close itself goes through CloseTask.)
- **Doctor data-integrity check.** `insight.go:762` `cmdDoctor` — add a check that flags done tasks which have a `claimed by` stamp but no `completed by` stamp (a broken calibration span), naming them, so the drift can never hide again.

## Scope (STRICT) — touch ONLY:
- `internal/store/store.go` (the CloseTask primitive)
- `internal/features/acceptance/acceptance.go`
- `internal/features/planning/planning.go`
- `internal/features/insight/insight.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the files above plus this task's file. `go build ./...` + `go test ./internal/...` green (calibration + planning have tests — the CloseTask refactor must not change the stamp text `completed by`). `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [ ] by-agent (and size) bands with n<10 are marked provisional and do not print a p10-p90 range as if calibrated
- [ ] accept and task done share one close primitive that always stamps 'completed by'; no path can close a task without the actuals stamp
- [ ] dacli doctor flags done tasks with a claim but no completion stamp (broken calibration spans)
- [ ] committed on branch by an agent; build + test green
## Log
- 2026-07-22T15:06:45Z claimed by a-yg7894d7yn
