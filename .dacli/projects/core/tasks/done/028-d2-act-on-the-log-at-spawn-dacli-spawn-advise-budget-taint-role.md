---
id: t-01KY4ZWW43218D6GGNA2XCPZQC
kind: task
created: 2026-07-22T13:23:01Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# D2: act on the log at spawn — dacli spawn --advise (budget, taint, role)
## Context
D2 turns the log from something you READ into something surfaced AT THE SPAWN DECISION — without dacli deciding for you (axiom 3: intelligence stays the model's). It builds on D1's calibration bands.

Anchors:
- `internal/features/execution/execution.go` `cmdSpawn` — add a `--advise` flag. After role/model/runtime/task are resolved but BEFORE minting identity/launching, print an advisory block and then continue the spawn normally (advice is additive; it never changes the decision):
  - **Budget/estimate from the band**: build the `store.Band{Role, Model, Runtime}` for this spawn, gather `store.CalibrationSamples(w)` whose `.Band` matches, and if there are enough (>=10, D1's threshold) print the empirical median + p10–p90 as the suggested sizing; else print "no band history yet". Reuse D1's helpers — do not duplicate the percentile math.
  - **Taint status**: report whether this task sits in a suspect blast radius. Reuse the existing taint machinery (`store.Taint` / `TaintResult.ExposedBriefs` in internal/store/taint.go) — e.g. if any tainted source's exposed briefs include this task's slug, warn "task NNN is in the blast radius of <source>"; else "taint: clean".
- `internal/features/insight/insight.go` `cmdNext` (~:113) — when suggesting work (esp. `--parallel`), surface a role hint from scope-matched lessons: use `store.WorkspaceLessons` (lessons.go:30) to note when a lesson's scope matches the task so the operator sees "lesson L applies — consider role R". Keep it a HINT line, not an automatic assignment.

## Scope (STRICT) — touch ONLY:
- `internal/features/execution/execution.go`
- `internal/features/insight/insight.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the two files above plus this task's file. `go build ./...` + `go test ./internal/...` green. `dacli note add finding` with the summary, then `dacli commit`. Box-checking is owner-only — file a completion finding, do not retry a refused check.

## Acceptance
- [x] spawn --advise prints a suggested --budget from the calibrated band before launching (advises, does not decide)
- [x] a task's taint status is shown before spawning a child on it
- [x] next --parallel role suggestion is influenced by scope-matched lessons
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T13:43:16Z completed by a-root
