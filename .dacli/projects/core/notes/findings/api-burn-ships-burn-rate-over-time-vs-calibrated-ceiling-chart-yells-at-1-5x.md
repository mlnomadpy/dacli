---
id: f-api-burn-ships-burn-rate-over-time-vs-calibrated-ceiling-chart-yells-at-1-5x
kind: note
note_kind: finding
created: 2026-07-24T11:08:52Z
created_by: a-bx9nvegpnc
about: [[144]]
severity: moderate
---
# /api/burn ships: burn-rate over time vs calibrated ceiling, chart yells at >=1.5x
internal/features/dashboard/burn.go: buildBurn() assembles (1) per-day run actuals from RunsDir usage.txt bucketed by the run's ULID-decoded start day (burnSeries + ulidTime), (2) calibrated role×model×runtime bands + a ceiling = median output_tokens across token-bearing store.CalibrationSamples, (3) live governor windowSpent per project from loop/*-governor.txt (burnWindows, parsed locally since dashboard cannot import orchestration — arch_test). Alert = latest-day per_run >= 1.5×ceiling (const AlertFactor). Registered at /api/burn (burnResponse envelope) AND embedded in /api/state.Burn so contracts can't drift. Handler test: burn_test.go TestAPIBurn/TestAPIStateEmbedsBurn/TestBurnEmptyWorkspaceIsZeroSafe/TestBurnNoAlertBelowThreshold/TestULIDTimeRoundTrip. UI: BurnRate.vue turns the panel danger-red + raises role=alert aria-live=assertive banner + hot bars at >=1.5x; wired into App.vue; TS strict + ESLint + prettier clean; 52 vitest tests green incl BurnRate.test.ts (4). go build + go test ./internal/... green.
