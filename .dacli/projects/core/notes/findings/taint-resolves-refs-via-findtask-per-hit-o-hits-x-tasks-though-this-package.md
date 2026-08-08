---
id: f-taint-resolves-refs-via-findtask-per-hit-o-hits-x-tasks-though-this-package
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-gkrbmb023t
about: [[t-01KY59FNDY9ECT40SDBF71VWBH]]
source_event: 01KY59P8YHH6MXXK7QPN9RYSY1
---
# Taint resolves refs via FindTask per hit (O(hits x tasks)) though this package already ships TaskIndex for exactly this loop
internal/store/taint.go:121 canonRef calls FindTask(w, ref) once per tainted hit (taint.go:77 for events, taint.go:101 for notes). FindTask (store.go:557) re-reads and re-parses the ENTIRE task tree on every call, so a Taint over N hits does N full task-tree reads = O(hits x tasks). This package already provides the fix it doesn't use: BuildTaskIndex/NewTaskIndex (store.go:580/589) read the tree once and resolve each ref O(1). Build one TaskIndex at the top of Taint and pass it to canonRef. Separately, ExposedBriefs (taint.go:149) calls ListTasks per exposed project inside the project loop — a single ListTasks(w,'',"") filtered in memory avoids re-walking overlapping projects. Extends the sibling FindTask-in-a-loop finding with the concrete in-store remedy.
