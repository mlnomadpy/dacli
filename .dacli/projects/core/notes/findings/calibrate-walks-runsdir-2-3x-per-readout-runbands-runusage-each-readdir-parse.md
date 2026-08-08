---
id: f-calibrate-walks-runsdir-2-3x-per-readout-runbands-runusage-each-readdir-parse
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-gkrbmb023t
about: [[t-01KY59FNDY9ECT40SDBF71VWBH]]
source_event: 01KY59MR5BQGY3TYC5XCZ2DC5Z
---
# calibrate walks RunsDir 2-3x per readout: runBands+runUsage each ReadDir+parse every invocation.txt
internal/store/calibration.go: CalibrationSamples (calibration.go:62) calls both runBands (calibration.go:173) and runUsage (calibration.go:103); each independently os.ReadDir(w.RunsDir()) and, per run dir, opens+scans invocation.txt (runUsage via runTaskID calibration.go:124, runBands at calibration.go:180). So the runs dir is walked twice and invocation.txt is parsed twice per run on every CalibrationSamples call. A single walk that reads task/role/model/runtime AND usage.txt in one pass would halve the I/O. Amplified in insight.go cmdEstimate which additionally calls TaskBand (a 3rd RunsDir walk) alongside CalibrationSamples for one task.
