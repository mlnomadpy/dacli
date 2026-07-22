---
id: t-01KY59FND2699RMWSC7MAK1MNV
kind: task
created: 2026-07-22T16:10:34Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# F2: act on token bands — spawn-time budget from measured token cost (advise + enforce)
## Context
The capstone of the calibration arc: F1 made dacli MEASURE cost in tokens (per role×model×runtime band); F2 makes it ACT on that at spawn. Builds directly on D2's `--advise` and F1's token calibration — do NOT add a new calibration walk; reuse the existing samples.

Anchors:
- `internal/features/execution/execution.go` — `printAdvisory(ctx, w, t, band)` (called from the `--advise` block at ~:270) currently shows a wall-clock/PERT sizing + taint. Enhance it: gather the band's token-bearing samples (`store.CalibrationSamples(w)` → filter to `.Band == band && .HasTokens()`), and when there are enough (reuse the n>=10 gate; below that mark PROVISIONAL, no hard number), compute an expected token cost = median `TokenRatio()` × the task's Te, and print a **suggested token budget** for this spawn. When the band has no token samples, keep today's wall-clock advice (honest fallback).
- **Spawn-time token gate.** Add `--max-tokens N`. After the advisory, if the band's expected token cost EXCEEDS N, refuse the spawn (clikit.Refusedf, exit 3) unless `--force` — exactly the shape of the D3 taint gate at ~:274 (advise DISPLAYS, the gate BLOCKS, --force overrides loudly). Below n>=10 the estimate is provisional: warn but do not hard-refuse on thin data.
- Add a small helper in `internal/store/calibration.go` if useful (e.g. a median-token-ratio-for-band over a []CalibSample) so the math isn't duplicated; keep `TokenRatio()`/`HasTokens()` as the primitives.

## Scope (STRICT) — touch ONLY:
- `internal/features/execution/execution.go`
- `internal/store/calibration.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the two files above (+ a test if useful) plus this task's file. `go build ./...` + `go test ./internal/...` green — a text-runtime spawn (no token data) must still advise on wall-clock and NOT refuse. `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [x] spawn --advise suggests a token budget from the role/model/runtime band's measured token-per-point (F1), not just wall-clock
- [x] an optional per-run token ceiling warns/refuses when a band's expected cost exceeds it
- [x] committed on branch by an agent; build + test green
## Log
- 2026-07-22T17:41:01Z claimed by a-q41r9cfexp
- 2026-07-22T17:46:56Z accepted by a-root
- 2026-07-22T17:46:56Z completed by a-root
