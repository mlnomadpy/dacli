---
id: 01KY59PTD63E3FBW9SE2YJ9S3K
kind: event
event_kind: finding
created: 2026-07-22T16:14:28Z
created_by: a-1s5dxes66y
about: [[t-01KY59FNEAAYDZ0PCTKE0HCBVA]]
origin: agent
applied: true
---
'dacli agents' reports ps lifetime-average %CPU as if it were current utilization

procmon.go:161 SampleGroup runs 'ps -A -o pgid=,pid=,rss=,%cpu=' and sums the %cpu column (:178), which on both Linux and macOS ps is cputime/elapsed AVERAGED over each process's whole lifetime, not an instantaneous sample. execution.go:1155 prints it as '%5.0f%% CPU', which reads as current utilization. Effect: a just-spawned agent doing heavy work under-reports (short elapsed, average still climbing), and a long-lived agent that spun hard early then went idle over-reports — the exact opposite of what an operator watching 'dacli agents' to spot a runaway needs. This is a reporting-honesty gap consistent with the project's 'GPU n/a, never faked' stance (gpuStr :1580). Fix: either label it 'avg %CPU (lifetime)', or take two ps snapshots a short interval apart and report the delta for a true instantaneous reading.
