---
id: t-01KY5GP5R2TF4MF5WS84KP18ZW
kind: task
created: 2026-07-22T18:16:27Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT2 R2: execution + procmon + calibration F-series stream-json
## Acceptance
- [ ] findings (file:line) in internal/features/execution/**, internal/procmon, internal/store/calibration.go. Focus on code changed since the last audit (G-series ghmirror, F1/F2 calibration+stream-json, ship/accept slices, G3 pr) and VERIFY the previous audit's fixes held (no regressions). File each issue as a finding with file:line, severity-tagged. Read-only — do not edit.
## Log
- 2026-07-22T18:16:35Z claimed by a-xaybvpxzth
- 2026-07-22T18:23:33Z finding by a-xaybvpxzth: runRecords clobbers a task's calibrated agent-band with a newer verify/supervise run's empty band (D1 regression) (event 01KY5GZXYK0WQJPG97DQM27DHW)
- 2026-07-22T18:23:33Z finding by a-xaybvpxzth: supervise/verify invocation.txt omit role/model, so recorded bands can never match the OrDash band used by the calibrate gate/advise (event 01KY5H0A4K0FBK3HNQ58TKKH33)
- 2026-07-22T18:23:33Z finding by a-xaybvpxzth: AUDIT2 R2 regression check: prior execution/procmon/stream-json fixes verified present, except calibration band join (event 01KY5H16GTE9EDQXJ6Z13WF0HJ)
