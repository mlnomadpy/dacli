---
id: 01KY5H16GTE9EDQXJ6Z13WF0HJ
kind: event
event_kind: finding
created: 2026-07-22T18:22:28Z
created_by: a-xaybvpxzth
about: [[t-01KY5GP5R2TF4MF5WS84KP18ZW]]
origin: agent
applied: true
---
AUDIT2 R2 regression check: prior execution/procmon/stream-json fixes verified present, except calibration band join

Re-read the four in-scope files against prior audit findings. HELD (verified in current source): (1) stdin+detach truncation — execRuntime detach path backs stdin with an unlinked *os.File temp (execution.go:912-923); (2) PID/PGID reuse — procmon.ProcStart/AliveIdentity/AliveRecord (procmon.go:149-191) and EVERY liveness probe funnels through AliveRecord (liveAgents:1597, cmdLogs:1434, cmdWait:1679); (3) teeStreamJSON usage loss — now bufio.Reader with no line cap + scanErr surfaced as '[dacli: usage capture incomplete]' (execution.go:1089-1111, 988-992); (4) detached raw-JSON transcript readability — renderStreamLine/renderTranscriptTo/lastTranscriptLine render on read (execution.go:1043-1096, 1446-1488), covering logs -f and agents --tail; (5) ps %cpu mislabeled — now 'CPUavg' with lifetime-average comment (execution.go:1301-1306, procmon.go:208); (6) calibrate multi-walk — single runRecords walk (calibration.go:160). REGRESSED: the single-walk refactor introduced the band-clobber (see major finding f-runrecords-clobbers...) — calibrate by-agent band goes empty for verified tasks. All other prior fixes hold; no new defects in verify.go/replay.go beyond the two calibration findings filed.
