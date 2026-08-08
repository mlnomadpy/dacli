---
id: f-taskband-re-scans-the-entire-runsdir-to-resolve-one-task-s-band-o-tasks-x-runs
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-gkrbmb023t
about: [[t-01KY59FNDY9ECT40SDBF71VWBH]]
source_event: 01KY59MYBFN0MWMKBFCQZHYEP9
---
# TaskBand re-scans the entire RunsDir to resolve one task's band — O(tasks x runs) if ever looped
internal/store/calibration.go:215 TaskBand(w, taskID) calls runBands(w) (calibration.go:173), which os.ReadDir+opens+scans EVERY run's invocation.txt, then discards all but one key. For a single estimate that is one full runs-tree read to answer one task; called per-task in any loop it becomes O(tasks x runs) disk reads — the same FindTask-in-a-loop shape siblings flagged for FindTask. Fix: expose a single-task lookup that stops at the first matching run, or have callers build the runBands map once and index into it (mirroring BuildTaskIndex in store.go:580).
