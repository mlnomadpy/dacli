---
id: t-01KY4ZWW3PCEN2T7RBERF6JNBN
kind: task
created: 2026-07-22T13:23:01Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 4, probable: 6, pessimistic: 10}
---
# D1: agent-native estimation — empirical role/model/runtime bands become the estimate
## Context
This is D1, the lead of the v2 "calibrate to agents not humans" direction (see the decision note). Today calibration bands actuals by SIZE only and reports a median multiplier beside human PERT. D1 bands by the AGENT — role × model × runtime — and makes the empirical distribution the estimate once a band has enough samples.

Anchors:
- `internal/store/calibration.go` — `CalibSample{Te, Hours}` and `CalibrationSamples(w)` join done tasks (Te + claim→completion wall-clock). Add a `Band` (role×model×runtime) to each sample by joining the done task to its run record: iterate `w.RunsDir()`, read each `<run>/invocation.txt` (has `task: <id>`, `role:`, `runtime:`, and — after your execution.go change — `model:`), and match `task:` to the sample's task ID. A task may have several runs (supervise turns); use the run that completed it (last matching run is fine). Actuals stay wall-clock (a time proxy — keep the existing honest caveat); tokens await runtime usage reporting, out of scope here.
- `internal/features/execution/execution.go:291` — the `invocation := fmt.Sprintf(...)` writes role/grant/runtime/binary but NOT model. Add `model: <modelName>` to the invocation record so the band is complete. `modelName` is already resolved in cmdSpawn. This is the ONLY change to execution.go.
- `internal/features/insight/insight.go` `cmdCalibrate` (~:485) — currently groups by the size `band()` closure. Add grouping by agent band (role×model×runtime): per band print `n`, median, and a p10–p90 spread (use `spm.Median`; add a percentile helper if needed, or compute p10/p90 from the sorted ratios). Keep the size-band view too. After `n>=10` in a band, mark that band's empirical distribution as the authoritative estimate.
- Surface it: `dacli estimate` (~:241) and/or the brief should show, for a task whose (role×model×runtime) band has n>=10, the empirical band distribution AS the estimate with human PERT labelled as the prior. Minimal acceptable: `dacli estimate` prints the empirical band line when the band qualifies. (Full brief wiring can be a follow-up; do the estimate readout at least.)

## Scope (STRICT) — touch ONLY:
- `internal/store/calibration.go`
- `internal/features/insight/insight.go`
- `internal/features/execution/execution.go` (ONLY the invocation.txt model line)

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the files above plus this task's file. `go build ./...` + `go test ./internal/...` green (cli TestMain clears DACLI_AGENT). Paste the summary as `dacli note add finding`, then `dacli commit`. Box-checking is owner-only — file a completion finding naming the branch; do not retry a refused `task check`.

## Acceptance
- [x] run records carry the model (added to the run record) so a done task's actuals can be banded by role x model x runtime
- [x] CalibrationSamples/calibrate group actuals by agent band (role x model x runtime): n, median, and p10-p90 spread per band
- [x] after n>=10 in a band, dacli estimate/brief surfaces the empirical band distribution AS the estimate; human PERT is shown as the prior
- [x] committed on branch by an agent; go build + go test ./internal/... green
## Log
- 2026-07-22T13:31:16Z completed by a-root
