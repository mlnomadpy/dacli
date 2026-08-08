---
id: d-144-api-burn-ceiling-is-the-calibrated-per-run-token-norm-not-the-governor
kind: note
note_kind: decision
created: 2026-07-24T11:03:03Z
created_by: a-bx9nvegpnc
about: [[144]]
---
# 144: /api/burn ceiling is the calibrated per-run token norm, not the governor window budget
## Chose
144: /api/burn ceiling is the calibrated per-run token norm, not the governor window budget
## Rejected
Make the governor WindowTokens budget the ceiling that yells
## Because
The persisted governor snapshot (loop/<project>-governor.txt) records window_spent + window_start but NOT the WindowTokens budget (orchestration.go:266 overloads loopState.WindowTokens with WindowSpent), so the window ceiling isn't reliably readable. The acceptance says 'yell at >=1.5x band', so ceiling = median output_tokens across token-bearing calibration samples (store.CalibrationSamples). Governor windowSpent is surfaced as informational per-project Windows[].
