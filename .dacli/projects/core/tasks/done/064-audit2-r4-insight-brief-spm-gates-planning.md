---
id: t-01KY5GP5S0V2MFQC2R2EFVEA5A
kind: task
created: 2026-07-22T18:16:27Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT2 R4: insight + brief + spm + gates + planning
## Acceptance
- [x] findings (file:line) in internal/features/{insight,planning,stagegate,briefing}, internal/brief, internal/spm, internal/gates. Focus on code changed since the last audit (G-series ghmirror, F1/F2 calibration+stream-json, ship/accept slices, G3 pr) and VERIFY the previous audit's fixes held (no regressions). File each issue as a finding with file:line, severity-tagged. Read-only — do not edit.
## Log
- 2026-07-22T18:16:35Z claimed by a-1zhjz6t2va
- 2026-07-22T18:23:33Z finding by a-1zhjz6t2va: estimate ignores the token-per-point unit that calibrate calls PREFERRED (event 01KY5GXKQH8YWABK3GPN1CRDY4)
- 2026-07-22T18:23:33Z finding by a-1zhjz6t2va: calibrate prints two contradictory AUTHORITATIVE 'IS the estimate' lines for one band (event 01KY5GXVN1H2FS2CD7Y01DTSVY)
- 2026-07-22T18:23:33Z finding by a-1zhjz6t2va: doctor analysis-paralysis double-counts a synced finding as both event and note (event 01KY5GY2CBFWZ42EZ7T13XG65S)
- 2026-07-22T18:23:33Z finding by a-1zhjz6t2va: brief 'What siblings found' scopes pending finding events to the task but shows notes project-wide (event 01KY5GY94188AXKXJK4P5NFQWR)
- 2026-07-22T18:23:33Z finding by a-1zhjz6t2va: R4 regression check: all prior in-scope fixes held (insight/brief/spm/gates) (event 01KY5GYNDWJANSWTPVRY2ERQJN)
- 2026-07-22T18:52:27Z accepted by a-root
- 2026-07-22T18:52:27Z completed by a-root
