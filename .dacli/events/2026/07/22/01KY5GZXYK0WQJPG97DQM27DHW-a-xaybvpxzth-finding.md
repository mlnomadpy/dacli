---
id: 01KY5GZXYK0WQJPG97DQM27DHW
kind: event
event_kind: finding
created: 2026-07-22T18:21:47Z
created_by: a-xaybvpxzth
about: [[t-01KY5GP5R2TF4MF5WS84KP18ZW]]
origin: agent
applied: true
---
runRecords clobbers a task's calibrated agent-band with a newer verify/supervise run's empty band (D1 regression)

internal/store/calibration.go:172-177 runRecords() overwrites rec.band UNCONDITIONALLY (rec.band = band) while the token usage join right below is GUARDED (if u, ok := readUsage(runDir); ok { rec.usage = u }). Run dirs are walked in ULID (chronological) order so the LAST run naming a task wins its band. But readInvocation (calibration.go:184) attributes a band to ANY run whose invocation.txt has a task: line — not just the completing spawn. dacli verify writes invocation.txt with keys run/verify_panel_seat/task/child/claim (verify.go:123-124) — NO role/model/runtime — so readInvocation returns an EMPTY Band{}. A verify run happens AFTER the finding it checks, so its ULID is newer than the completing spawn's; runRecords therefore clobbers the real band to Band{} for that task. Effect: LoadCalibration sets CalibSample.Band=Band{} and TaskBand()==false for every VERIFIED task, so calibrate by-agent band and spawn --advise/--max-tokens go empty again — the D1/E3 regression f-calibrate-by-agent-band-stays-empty was shipped to fix. Same root cause pollutes the token join. Fix: overwrite band only when !band.Empty() (mirror the guarded usage join), or key runRecords on the run that completed the task.
