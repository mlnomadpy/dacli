---
id: d-compute-p10-p90-with-a-local-percentile-in-insight-go-not-a-spm-helper
kind: note
note_kind: decision
created: 2026-07-22T13:29:14Z
created_by: a-sr9e6xf1d0
about: [[027]]
---
# Compute p10/p90 with a local percentile() in insight.go, not a spm helper
## Chose
Compute p10/p90 with a local percentile() in insight.go, not a spm helper
## Rejected
Add Percentile() to internal/spm/estimate.go beside Median()
## Because
The task scope is STRICT to calibration.go/insight.go/execution.go; spm/estimate.go is out of scope. The percentile is only needed by the calibrate/estimate readouts, so a small linear-interpolation helper local to insight.go keeps the change inside the allowed files without a cross-package edit.
