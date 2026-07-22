---
id: t-01KY5GP5QJS16DCPAHQMTFBE5X
kind: task
created: 2026-07-22T18:16:27Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT2 R1: ghmirror G-series + selfreport
## Acceptance
- [ ] findings (file:line) in internal/features/ghmirror/** + selfreport. Focus on code changed since the last audit (G-series ghmirror, F1/F2 calibration+stream-json, ship/accept slices, G3 pr) and VERIFY the previous audit's fixes held (no regressions). File each issue as a finding with file:line, severity-tagged. Read-only — do not edit.
## Log
- 2026-07-22T18:16:35Z claimed by a-drg65wknjt
- 2026-07-22T18:23:33Z finding by a-drg65wknjt: Disclosure-gate consent is a bare boolean, not scoped to the consented repo (event 01KY5GWHXB0B20Z59Z48RNY8KY)
- 2026-07-22T18:23:33Z finding by a-drg65wknjt: searchByMarker does a full gh issue list per task and per decision inside the push loop (event 01KY5GWS9XB399MGCY74Y7YC9D)
- 2026-07-22T18:23:33Z finding by a-drg65wknjt: cmdPush rewrites every task file on every push even when the mapping is unchanged (event 01KY5GX01M2EP2B3TFDVFTWTYE)
- 2026-07-22T18:23:33Z finding by a-drg65wknjt: findingAboutTask matches the task by loose substring, missing unpadded refs and risking cross-matches (event 01KY5GX93ZYNKVMPYXR5KAQZD2)
- 2026-07-22T18:23:33Z finding by a-drg65wknjt: dacli report attaches workspace name and raw transcript tail to the public upstream repo with no disclosure gate (event 01KY5GXGX5MER45S1MMDNB3JV4)
- 2026-07-22T18:23:33Z finding by a-drg65wknjt: AUDIT2 R1 regression check: prior ghmirror/selfreport fixes held (event 01KY5GXTP25SN8KKTSF4G4CMFF)
