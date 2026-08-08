---
id: 01KYFGJ0CTBXQ1X7T3PA6DKWBX
kind: event
event_kind: finding
created: 2026-07-26T15:26:35Z
created_by: a-q2y900ts5s
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: true
---
burn alert dilutes per-run rate: Series counts all runs, Ceiling only completing runs (false-negative yell)

internal/features/dashboard/burn.go: the /api/burn overspend alert compares two metrics computed over DIFFERENT run populations, so an over-budget implementer run can be silently masked. burnSeries (burn.go:121-163) sums output_tokens AND run COUNT across EVERY usage-bearing run dir per day (readRunUsage at :131 applies no verify/role filter), then Rate = latest day's PerRun = tokens/runs (:158, :102). But calibratedCeiling (:266-274) is the median over store.CalibrationSamples, which is ONE sample per DONE task from its completing run (calibration.go:157-176) and EXCLUDES verify-panel seats (calibration.go:217 skips isVerify) and review-role runs (which never complete a task). Ratio=Rate/Ceiling and Alert=Ratio>=1.5 (:108-109) therefore threshold a per-ANY-run average against a per-COMPLETING-run norm. Failure scenario: ceiling=20k/run; latest day = one 40k implementer run (2x over) + 3 cheap review/verify runs at 2k each -> Runs=4, tokens=46k, PerRun=11.5k, Ratio=0.575 -> NO alert, even though an implementer run burned 2x the norm. Cheap same-day runs dilute PerRun and suppress the overspend yell. Fix direction: make burnSeries count the same population the ceiling is built from (completing/implementer runs, excluding verify seats and review-role runs) OR compute a per-completing-run rate, so Rate and Ceiling are apples-to-apples.
