---
id: t-01KYFGJ81Q5XKKN97NTNBD9SJ3
kind: task
created: 2026-07-26T15:26:43Z
created_by: a-q2y900ts5s
owner: a-q2y900ts5s
priority: should
---
# burn alert: compute per-run Rate over the same population as the calibrated Ceiling (fix false-negative overspend yell)
## Acceptance
- [ ] burnSeries (or a sibling) computes the day's PerRun rate over the SAME run population the ceiling is built from: completing/implementer runs only, excluding verify-panel seats and review-role runs, matching store.CalibrationSamples
- [ ] A unit test in burn_test.go proves the false-negative is closed: a day with one over-ceiling implementer run plus several cheap review/verify runs yields Ratio>=1.5 and Alert=true (today it dilutes to Alert=false)
- [ ] The burnPoint series shown on the chart and the Rate/Ratio/Alert fields remain internally consistent (documented which population each counts); go build ./... and go test ./internal/... stay green
- [ ] No change to the calibration or governor slices; the fix stays inside internal/features/dashboard
## Log
