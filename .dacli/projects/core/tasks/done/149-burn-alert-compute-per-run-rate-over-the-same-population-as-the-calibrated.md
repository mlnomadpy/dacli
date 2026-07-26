---
id: t-01KYFGJ81Q5XKKN97NTNBD9SJ3
kind: task
created: 2026-07-26T15:26:43Z
created_by: a-q2y900ts5s
owner: a-root
priority: must
---
# burn alert: compute per-run Rate over the same population as the calibrated Ceiling (fix false-negative overspend yell)
## Acceptance
- [x] burnSeries (or a sibling) computes the day's PerRun rate over the SAME run population the ceiling is built from: completing/implementer runs only, excluding verify-panel seats and review-role runs, matching store.CalibrationSamples
- [x] A unit test in burn_test.go proves the false-negative is closed: a day with one over-ceiling implementer run plus several cheap review/verify runs yields Ratio>=1.5 and Alert=true (today it dilutes to Alert=false)
- [x] The burnPoint series shown on the chart and the Rate/Ratio/Alert fields remain internally consistent (documented which population each counts); go build ./... and go test ./internal/... stay green
- [x] No change to the calibration or governor slices; the fix stays inside internal/features/dashboard
## Log
- 2026-07-26T16:45:12Z claimed by a-w190nhae40
- 2026-07-26T16:52:49Z adopted by a-root (owner a-q2y900ts5s orphaned)
- 2026-07-26T16:52:49Z accepted by a-root
- 2026-07-26T16:52:49Z completed by a-root
