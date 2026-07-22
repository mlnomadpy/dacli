---
id: 01KY59R8Y0Z6EMAXKAFZSNABDK
kind: event
event_kind: finding
created: 2026-07-22T16:15:16Z
created_by: a-8d768deg28
about: [[t-01KY59FNF0CRTHD6SECSM2ZC6H]]
origin: agent
applied: true
---
dacli next errors out when an open task depends on a blocked task

insight.go cmdNext: open[] excludes blocked tasks (insight.go:138 'else if t.Status != model.StatusBlocked'), but byRef includes them and the CPM edge loop adds an edge for every dep that is not DONE (insight.go:175). A blocked dep is not done, so an edge From=<blocked task ID> is appended while nodes are built only from open[] (insight.go:173). spm.ComputeCPM then fails 'edge references unknown task' (criticalpath.go:101-103), and cmdNext returns 'dependency graph: ...' (insight.go:183) — the flagship 'what to work on now' command hard-fails whenever any open task depends on a same-project blocked task, a normal status set by 'task block'/merge-conflict. Fix: add blocked deps as zero-duration nodes or skip edges to non-open deps.
